package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/harrybrwn/config"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/app"
	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		log.Println(err)
	}
}

func run() error {
	var conf = &app.Config{}
	config.SetFilename("mt.yml")
	config.SetType("yml")
	config.AddPath(".")
	config.SetConfig(conf)
	if err := config.ReadConfigFile(); err != nil {
		log.Println("Warning:", err)
	}
	if err := conf.Init(); err != nil {
		return err
	}
	gin.SetMode(config.GetString("mode"))

	db, err := sqlx.Connect(conf.Database.Driver, conf.GetDSN())
	if err != nil {
		return errors.Wrap(err, "could not open db")
	}
	defer db.Close()

	r := gin.New()
	r.Use(gin.Recovery(), gin.LoggerWithConfig(app.LoggerConfig))
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, app.ErrStatus(404, "no route for "+c.Request.URL.Path))
	})

	a := app.App{
		DB:     db,
		Config: conf,
		Engine: r,
	}

	auth, err := a.NewJWTAuth()
	if err != nil {
		return errors.Wrap(err, "could not create auth middleware")
	}
	if err = auth.MiddlewareInit(); err != nil {
		return errors.Wrap(err, "could not init auth middleware")
	}

	v1 := r.Group("/api/v1")
	if config.GetString("mode") == "debug" || true {
		// CORS
		v1.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Next()
		})
	}
	a.RegisterRoutes(v1)

	r.POST("/graphql", a.GraphQLHander())
	r.GET("/graphql/playground", a.GraphQLPlayground("/graphql"))

	r.POST("/login", auth.LoginHandler)
	r.GET("/admin", auth.MiddlewareFunc(), func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"success": "yay",
		})
	})

	v1.GET("/debug",
		auth.MiddlewareFunc(),
		func(c *gin.Context) {
			data := map[string]interface{}{
				"time":    time.Now(),
				"testing": true,
				"mode":    conf.Mode,
				"db": map[string]interface{}{
					"dsn":    conf.GetDSN(),
					"driver": conf.Database.Driver,
				},
			}
			c.IndentedJSON(200, data)
		},
	)

	addr := conf.Address()
	fmt.Printf("\n\nRunning on \x1b[32;4mhttp://%s\x1b[0m\n", addr)

	srv := http.Server{
		Addr:           addr,
		Handler:        &a,
		ReadTimeout:    time.Minute * 5,
		WriteTimeout:   time.Minute * 5,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
	}
	return srv.ListenAndServe()
}
