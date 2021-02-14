package app

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func TestWebsockets(t *testing.T) {
	conf := testConfig()
	a := &App{
		DB:        sqlx.MustConnect(conf.Database.Driver, conf.GetDSN()),
		Config:    conf,
		RateStore: memory.NewStore(),
		Protected: func(c *gin.Context) { c.Next() },
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a.Engine = r
	db.Set(a.DB)

	ch := make(chan interface{})
	r.POST("/update", a.wsPublisher(ch))
	r.GET("/updates", a.wsSub(ch))

	srv := httptest.NewServer(a)
	u, err := url.Parse(srv.URL)

	u.Path = "/updates"
	u.Scheme = "ws"
	wsconn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 101 {
		t.Error("expected 101")
	}

	done := make(chan struct{})
	go func() {
		for {
			_, p, err := wsconn.ReadMessage()
			if err != nil {
				t.Error(err)
			}
			if len(p) == 0 {
				t.Error("websocket responded with no data")
			}
			done <- struct{}{}
			return
		}
	}()

	type update struct {
		CRN       int       `json:"crn"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	b := bytes.Buffer{}
	if err = json.NewEncoder(&b).Encode(&update{10, time.Now()}); err != nil {
		t.Fatal(err)
	}
	resp, err = srv.Client().Post(
		srv.URL+"/update",
		"application/json",
		&b,
	)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Error("bad status code")
	}
	<-done

	if err = wsconn.Close(); err != nil {
		t.Error(err)
	}
	srv.Close()
}
