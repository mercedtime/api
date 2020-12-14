package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mercedtime/api/db/models"
)

func testConfig() *Config {
	conf := &Config{}
	conf.Database.Driver = "postgres"
	conf.Database.Host = "localhost"
	conf.Database.Port = 25432
	conf.Database.User = env("POSTGRES_USER", "mt")
	conf.Database.Password = env("POSTGRES_PASSWORD", "test")
	conf.Database.Name = env("POSTGRES_DB", "mercedtime")
	conf.Database.SSL = "disable"
	return conf
}

func testApp(t *testing.T) *App {
	t.Helper()
	conf := testConfig()
	a := &App{
		DB:     sqlx.MustConnect(conf.Database.Driver, conf.GetDSN()),
		Config: conf,
	}
	gin.SetMode(gin.TestMode)
	a.Engine = gin.New()
	a.RegisterRoutes(&a.Engine.RouterGroup)
	return a
}

func TestListEndpoints(t *testing.T) {
	app := testApp(t)
	for _, tst := range []struct {
		Path          string
		Limit, Offset int
		ShouldFail    bool
		Code          int
		Query         url.Values
	}{
		{Path: "/lectures", Limit: 10, Offset: 12, Code: 200},
		{Path: "/lectures", Query: url.Values{"subject": {"bio"}}, Code: 200},
		{
			Path:  "/lectures",
			Limit: 2, Offset: -1,
			Code: 500,
		},
		{Path: "/labs", Limit: 4, Offset: 0, Code: 200},
		{Path: "/labs", Limit: 4, Offset: -1, Code: 500},
		{Path: "/discussions", Limit: 3, Code: 200},
		{Path: "/discussions", Limit: 3, Offset: 3, Code: 200},
		{Path: "/discussions", Limit: 3, Offset: -1, Code: 500},
		{Path: "/exams", Limit: 12, Code: 200},
		{Path: "/exams", Limit: 12, Offset: 8, Code: 200},
		{Path: "/exams", Limit: 12, Offset: -1, Code: 500},
		{Path: "/instructors", Limit: 2, Code: 200},
		{Path: "/instructors", Limit: 2, Offset: 12, Code: 200},
		{Path: "/instructors", Limit: 2, Offset: -1, Code: 500},
	} {
		r := &http.Request{
			Method: "GET",
			Proto:  "HTTP/1.1",
			URL: &url.URL{
				Path: tst.Path,
			},
		}
		if tst.Code == 0 {
			tst.Code = 200
		}
		checkLim := len(tst.Query) == 0
		if checkLim {
			tst.Query = url.Values{}
			tst.Query.Set("offset", strconv.FormatInt(int64(tst.Offset), 10))
			tst.Query.Set("limit", strconv.FormatInt(int64(tst.Limit), 10))
		}
		r.URL.RawQuery = tst.Query.Encode()

		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)

		if w.Code != tst.Code {
			t.Errorf("bad status code, got %d, want %d", w.Code, tst.Code)
			continue
		}
		if tst.Code >= 300 {
			continue // dont need to check the result
		}
		list := make([]interface{}, 0)
		if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
			t.Error(err)
			continue
		}
		if checkLim && len(list) != tst.Limit {
			t.Errorf("expected response of length %d, got %d", tst.Limit, len(list))
		}
	}
}

func TestPostUser(t *testing.T) {
	a := testApp(t)
	ts := httptest.NewServer(a.Engine)
	defer ts.Close()
	resp, err := ts.Client().Post(ts.URL+"/user", "application/json", strings.NewReader(`
		{"name":"testuser"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Error("wrong status code")
	}
	m := make(map[string]interface{})
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
}

func TestListEndpointsServer(t *testing.T) {
	app := testApp(t)
	ts := httptest.NewServer(app.Engine)
	defer ts.Close()
	for _, e := range []string{
		"/lectures?limit=10&offset=2",
		"/lectures?limit=100&offset=7&subject=bio",
		"/exams?limit=2&offset=4",
		"/labs?limit=10&offset=3",
		"/discussions?limit=20&offset=4",
	} {
		resp, err := ts.Client().Get(ts.URL + e)
		if err != nil {
			t.Error(err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("\"%s\" from %s", resp.Status, e)
		}
		if err = resp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestLecture(t *testing.T) {
	app := testApp(t)
	ts := httptest.NewServer(app.Engine)
	defer ts.Close()
	resp, err := ts.Client().Get(ts.URL + "/lectures?limit=30&offset=30&subject=cse")
	if err != nil {
		t.Fatal(err)
	}
	var (
		lectures []*models.Lecture
	)
	if err = json.NewDecoder(resp.Body).Decode(&lectures); err != nil {
		t.Fatal(err)
	}
	if err = resp.Body.Close(); err != nil {
		t.Error(err)
	}
	for _, lecture := range lectures {
		var lect models.Lecture
		resp, err = ts.Client().Get(ts.URL + fmt.Sprintf("/lecture/%d", lecture.CRN))
		if err != nil {
			t.Error(err)
		}
		if err = json.NewDecoder(resp.Body).Decode(&lect); err != nil {
			t.Error(err)
		}
		if err = resp.Body.Close(); err != nil {
			t.Error(err)
		}
		if lect.CRN != lecture.CRN {
			t.Error("wrong crn")
		}
		if lect.Days != lecture.Days {
			t.Error("wrong days")
		}
		if lect.InstructorID != lecture.InstructorID {
			t.Error("wrong instructor id")
		}
		if lect.Units != lecture.Units {
			t.Error("wrong units")
		}
	}
}

func TestGetLabs(t *testing.T) {
	app := testApp(t)
	ts := httptest.NewServer(app.Engine)
	defer ts.Close()

	var labs []models.LabDisc
	resp, err := http.Get(ts.URL + "/labs?limit=10")
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&labs); err != nil {
		t.Error(err)
	}
	if len(labs) != 10 {
		t.Error("wrong number of labs")
	}
	for _, l := range labs {
		if l.CRN == 0 {
			t.Error("got zero crn")
		}
	}
}

func env(name string, deflt ...string) string {
	e := os.Getenv(name)
	if e == "" {
		for _, v := range deflt {
			if v != "" {
				return v
			}
		}
	}
	return e
}
