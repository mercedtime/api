package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
	"github.com/mercedtime/api/users"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func Test(t *testing.T) {
	// a := testApp(t)
	// rows, err := a.DB.Query("SELECT updated_at FROM aux")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// for rows.Next() {
	// 	tm := time.Time{}
	// 	if err = rows.Scan(&tm); err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	fmt.Println(tm)
	// }
}

func testConfig() *Config {
	return &Config{
		InMemoryRateStore: true,
		Database: DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     25432,
			User:     env("POSTGRES_USER", "mt"),
			Password: env("POSTGRES_PASSWORD", "test"),
			Name:     env("POSTGRES_DB", "mercedtime"),
			SSL:      "disable",
		},
	}
}

func testApp(t *testing.T) *App {
	t.Helper()
	conf := testConfig()
	a := &App{
		DB:        sqlx.MustConnect(conf.Database.Driver, conf.GetDSN()),
		Config:    conf,
		RateStore: memory.NewStore(),
		Protected: func(c *gin.Context) { c.Next() },
	}
	gin.SetMode(gin.TestMode)
	a.Engine = gin.New()
	a.RegisterRoutes(&a.Engine.RouterGroup)
	return a
}

func TestListEndpoints(t *testing.T) {
	app := testApp(t)
	defer app.Close()
	for _, tst := range []struct {
		Path          string
		Limit, Offset int
		Code          int
		Query         url.Values
		Subject       string
	}{
		{Path: "/lectures", Limit: 10, Offset: 12, Code: 200},
		{Path: "/lectures", Query: url.Values{"subject": {"bio"}}, Code: 200},
		{Path: "/lectures", Limit: 2, Offset: -1, Code: 400},
		{Path: "/labs", Limit: 4, Offset: 0, Code: 200},
		{Path: "/labs", Limit: 4, Offset: -1, Code: 400},
		{Path: "/discussions", Limit: 3, Code: 200},
		{Path: "/discussions", Limit: 3, Offset: 3, Code: 200},
		{Path: "/discussions", Limit: 3, Offset: -1, Code: 400},
		{Path: "/exams", Limit: 12, Code: 200},
		{Path: "/exams", Limit: 12, Offset: 8, Code: 200},
		{Path: "/exams", Limit: 12, Offset: -1, Code: 400},
		{Path: "/instructors", Limit: 2, Code: 200},
		{Path: "/instructors", Limit: 2, Offset: 12, Code: 200},
		{Path: "/instructors", Limit: 2, Offset: -1, Code: 400},
		{Path: "/courses", Limit: 30, Offset: 2, Code: 200},
		{Path: "/courses", Subject: "cse", Code: 200},
		{Path: "/courses", Query: url.Values{
			"subject": {"cse"},
			"year":    {"2021"},
			"term":    {"spring"},
			"limit":   {"53"},
		}, /* Subject: "cse",*/ Code: 200},
		{Path: "/catalog/2021/spring", Code: 200, Limit: 23},
		{Path: "/catalog/2021/spring", Code: 200, Query: url.Values{"subject": {"anth"}, "limit": {"2"}}},
		{Path: "/catalog/2020/fall", Code: 200, Limit: 3, Offset: 5},
		{Path: "/catalog/2020/summer", Code: 200, Limit: 3, Offset: 5},
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
			if tst.Subject != "" {
				tst.Query.Set("subject", tst.Subject)
			}
		}
		r.URL.RawQuery = tst.Query.Encode()
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)

		if w.Code != tst.Code {
			t.Errorf("%s: bad status code, got %d, want %d", r.URL, w.Code, tst.Code)
			t.Log(w.Body.String())
			continue
		}
		if tst.Code >= 300 {
			continue // dont need to check the result
		}
		list := make([]map[string]interface{}, 0)
		if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
			t.Error(err)
			continue
		}
		if tst.Subject != "" {
			for _, item := range list {
				for k, v := range item {
					if k != "subject" {
						fmt.Println(k, v)
						continue
					}
					if strings.ToLower(v.(string)) != strings.ToLower(tst.Subject) {
						t.Errorf("expected subject %s, got %s from %s", tst.Subject, v, r.URL)
					} else {
						fmt.Println(v, tst.Subject)
					}
				}
			}
		}
		if checkLim && len(list) != tst.Limit {
			t.Errorf("expected response of length %d, got %d", tst.Limit, len(list))
		}
	}
}

func TestLectureRoutes(t *testing.T) {
	t.Skip("not actually using these")
	var (
		crn int
		app = testApp(t)
	)
	defer app.Close()
	// get a testing crn that has an exam
	row := app.DB.QueryRow(`
		select l.crn from lectures l, exam e
		where l.crn = e.crn
		order by random() limit 1`)
	if err := row.Scan(&crn); err != nil {
		t.Fatal(err)
	}

	for _, tst := range []struct {
		Path string
		Code int
	}{
		{Path: fmt.Sprintf("/lecture/%d", crn), Code: 200},
		{Path: "/lecture/9999999", Code: 404},
		{Path: fmt.Sprintf("/lecture/%d/exam", crn), Code: 200},
		{Path: "/lecture/9999999/exam", Code: 404},
		{Path: fmt.Sprintf("/lecture/%d/labs", crn), Code: 200},
		// {Path: "/lecture/9999999/labs", Code: 404}, // TODO this should not return 200
		{Path: fmt.Sprintf("/lecture/%d/instructor", crn), Code: 200},
		{Path: "/lecture/9999999/instructor", Code: 200}, // TODO this return 404
	} {
		r := &http.Request{
			Method: "GET",
			Proto:  "HTTP/1.1",
			URL:    &url.URL{Path: tst.Path},
		}
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		if tst.Code != w.Code {
			t.Log(w.Body.String())
			t.Errorf("'%s' bad status code: got %d, want %d", tst.Path, w.Code, tst.Code)
			continue
		}
	}
}

func TestInstructorRoutes(t *testing.T) {
	var (
		id  int
		app = testApp(t)
	)
	defer app.Close()
	row := app.DB.QueryRow(`
		select id from instructor
		order by random() limit 1`) // get a random crn
	if err := row.Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, tst := range []struct {
		Path  string
		Code  int
		Query url.Values
	}{
		{Path: fmt.Sprintf("/instructor/%d", id)},
		{Path: "/instructor/999999", Code: 404},
		// {Path: fmt.Sprintf("/instructor/%d/courses", id)},
		// {Path: "/instructor/999999/courses", Code: 404},
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

		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)

		if w.Code != tst.Code {
			t.Errorf("'%s' bad status code: got %d, want %d", tst.Path, w.Code, tst.Code)
			continue
		}
	}
}

func TestUser(t *testing.T) {
	conf := testConfig()
	a := &App{
		DB:        sqlx.MustConnect(conf.Database.Driver, conf.GetDSN()),
		Config:    conf,
		RateStore: memory.NewStore(),
		Protected: func(c *gin.Context) { c.Next() },
	}
	gin.SetMode(gin.TestMode)
	a.Engine = gin.New()
	auth, err := a.NewJWTAuth()
	if err != nil {
		t.Fatal(err)
	}
	// regegister routes with new auth
	a.RegisterRoutes(&a.Engine.RouterGroup)
	a.Engine.POST("/login", auth.LoginHandler)
	ts := httptest.NewServer(a.Engine)
	defer a.Close()
	defer ts.Close()

	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tst := range []struct {
		Path string `json:"-"`
		Data string `json:"-"`
		Code int    `json:"-"`

		Name     string `json:"name"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}{
		{Path: "/user", Data: `{"name":"testuser"}`, Code: 400},
		{Path: "/user", Name: "testuser", Email: "test@test.com", Password: "password2", Code: 201},
		{Path: "/user", Name: "testuser", Email: "invalidemail", Password: "password2", Code: 201},
	} {
		var (
			body io.Reader
			b    bytes.Buffer
		)
		if tst.Data == "" {
			if err = json.NewEncoder(&b).Encode(&tst); err != nil {
				t.Fatal(err)
			}
			body = &b
		} else {
			body = strings.NewReader(tst.Data)
		}
		resp, err := ts.Client().Post(ts.URL+tst.Path, "application/json", body)
		if err != nil {
			t.Error(err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != tst.Code {
			t.Errorf("expected status %d; got %d", tst.Code, resp.StatusCode)
			continue
		}
		if tst.Code >= 300 {
			continue
		}
		var (
			u     users.User
			user  *users.User
			token = map[string]interface{}{"token": ""}
		)
		buildReq := func(m string) *http.Request {
			return &http.Request{
				Method: m, Proto: "HTTP/1.1",
				URL: url, Header: http.Header{
					"Authorization": {fmt.Sprintf("Bearer %s", token["token"])},
				},
			}
		}
		if err = json.NewDecoder(resp.Body).Decode(&u); err != nil {
			t.Error(err)
			continue
		}
		if tst.Name == "" || tst.Email == "" {
			goto Cleanup
		}

		user, err = a.GetUser(users.User{Name: tst.Name, Email: tst.Email})
		if err != nil {
			t.Error(err)
		}
		if user.Name != u.Name {
			t.Errorf("username response differs from database username; database: %s, response: %s", user.Name, u.Name)
		}

		b.Reset()
		if err = json.NewEncoder(&b).Encode(tst); err != nil {
			t.Error(err)
		}
		resp, err = ts.Client().Post(ts.URL+"/login", "application/json", &b)
		if err != nil {
			t.Error(err)
			goto Cleanup
		}
		if err = json.NewDecoder(resp.Body).Decode(&token); err != nil {
			t.Error(err)
			goto Cleanup
		}

		url.Path = "/user/self"
		if resp, err := ts.Client().Do(buildReq("GET")); err != nil {
			t.Error(resp.Status, err)
		}
		url.Path = "/user/badid"
		if resp, err = ts.Client().Do(buildReq("GET")); err == nil {
			if resp.StatusCode < 300 {
				t.Errorf("expected a bad status, got %s", resp.Status)
			}
		} else {
			t.Error(err)
		}
		url.Path = fmt.Sprintf("/user/%d", u.ID)
		resp, err = ts.Client().Do(buildReq("GET"))
		if err != nil {
			t.Error(err)
			goto Cleanup
		}
		if err = json.NewDecoder(resp.Body).Decode(user); err != nil {
			t.Error(err)
		}
		if user.Name != u.Name {
			t.Errorf("username response differs from database username; database: %s, response: %s", user.Name, u.Name)
		}
		resp.Body.Close()

	Cleanup:
		url.Path = "/user/200000000"
		resp, err = ts.Client().Do(buildReq("DELETE"))
		if err != nil {
			t.Error(err)
		}
		if resp.StatusCode < 300 {
			t.Error("should be unauthorized")
		}
		resp.Body.Close()

		url.Path = fmt.Sprintf("/user/%d", u.ID)
		resp, err = ts.Client().Do(buildReq("DELETE"))
		if err != nil {
			t.Error(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("did not delete user; %s", resp.Status)
		}
	}
	if _, err := a.GetUser(users.User{}); err == nil {
		t.Error("exptected an error from getting an empty user type")
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
			// b, _ := ioutil.ReadAll(resp.Body)
			// fmt.Printf("%s\n", b)
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
	defer app.Close()
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
		if lect.InstructorID != lecture.InstructorID {
			t.Error("wrong instructor id")
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
