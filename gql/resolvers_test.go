package gql

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/db"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func testApp(t *testing.T) *app.App {
	t.Helper()
	conf := testConfig()
	a := &app.App{
		DB:        sqlx.MustConnect(conf.Database.Driver, conf.GetDSN()),
		Config:    conf,
		RateStore: memory.NewStore(),
		Protected: func(c *gin.Context) { c.Next() },
	}
	gin.SetMode(gin.TestMode)
	a.Engine = gin.New()
	a.RegisterRoutes(&a.Engine.RouterGroup)
	db.Set(a.DB)
	return a
}

func Test(t *testing.T) {}

func testConfig() *app.Config {
	return &app.Config{
		InMemoryRateStore: true,
		Database: app.DatabaseConfig{
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
