package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/harrybrwn/config"
	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/gql"
	"github.com/sirupsen/logrus"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/ulule/limiter/v3"
	ginlimit "github.com/ulule/limiter/v3/drivers/middleware/gin"
)

func main() {
	if err := run(); err != nil {
		log.Println(err)
	}
}

// Config is the config struct
type Config struct {
	app.Config `yaml:",inline"`

	TLS      bool   `yaml:"tls"`
	CertFile string `yaml:"cert"`
	KeyFile  string `yaml:"key"`
}

func (c *Config) setup() error {
	config.SetFilename("mt.yml")
	config.SetType("yml")
	config.AddPath(".")
	config.SetConfig(c)
	return c.Init()
}

func run() error {
	var conf = &Config{}
	if err := conf.setup(); err != nil {
		return err
	}

	conf.InMemoryRateStore = true
	r := gin.New()
	r.Use(gin.Recovery(), gin.LoggerWithConfig(app.LoggerConfig))
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, app.ErrStatus(404, "no route for "+c.Request.URL.Path))
	})

	a, err := app.New(&conf.Config)
	if err != nil {
		return err
	}
	defer a.Close()
	a.Engine = r

	logrus.SetOutput(os.Stdout)
	l := logrus.New()
	l.Out = os.Stdout
	logrus.Debug("testing logger")

	auth, err := a.NewJWTAuth()
	if err != nil {
		return errors.Wrap(err, "could not create auth middleware")
	}

	r.Use(ginlimit.NewMiddleware(limiter.New(
		a.RateStore,
		limiter.Rate{
			Period: time.Second,
			Limit:  5,
		}),
	))
	v1 := r.Group("/api/v1")
	if config.GetString("mode") == "debug" || true {
		r.Use(cors)
		v1.Use(cors)
	}
	v1.GET("/test", auth.MiddlewareFunc(), func(c *gin.Context) {
		fmt.Println(c)
	})
	v1.OPTIONS("/user", func(c *gin.Context) { c.Status(204) })
	a.RegisterRoutes(v1)

	r.POST("/graphql", gql.Handler(a.DB))
	r.GET("/graphql/playground", gql.Playground("/graphql"))

	v1.OPTIONS("/auth/login", func(c *gin.Context) { c.Status(204) })
	v1.POST("/auth/login", auth.LoginHandler)

	v1.OPTIONS("/auth/logout", func(c *gin.Context) { c.Status(204) })
	v1.POST("/auth/logout", auth.LogoutHandler)
	v1.GET("/auth/logout", auth.LogoutHandler)

	v1.OPTIONS("/auth/refresh", func(c *gin.Context) { c.Status(204) })
	v1.GET("/auth/refresh", auth.RefreshHandler)

	r.OPTIONS("/signup", func(c *gin.Context) { c.Status(204) })
	r.POST("/signup", a.SilentCreateUser, auth.LoginHandler)

	r.GET("/admin", auth.MiddlewareFunc(), func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"success": "yay",
		})
	})

	logrus.Debug("testing logger")
	return listen(conf, a)
}

func listen(conf *Config, h http.Handler) error {
	var addr = net.JoinHostPort(conf.Host, strconv.FormatInt(conf.Port, 10))

	cert, err := tls.LoadX509KeyPair(conf.CertFile, conf.KeyFile)
	if err != nil {
		log.Printf("Warning: %v\n", err)
		conf.TLS = false
	}
	srv := http.Server{
		Addr:           addr,
		Handler:        h,
		ReadTimeout:    time.Minute * 5,
		WriteTimeout:   time.Minute * 5,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
		TLSConfig: &tls.Config{
			ServerName:   "mt",
			Certificates: []tls.Certificate{cert},
		},
		// TLSNextProto: map[string]func(s *http.Server, conn *tls.Conn, h http.Handler){},
	}

	fmt.Printf("\n\nRunning on ")
	if conf.TLS {
		fmt.Printf("\x1b[32;4mhttps://%s\x1b[0m\n", addr)
		return srv.ListenAndServeTLS("", "")
	}
	fmt.Printf("\x1b[32;4mhttp://%s\x1b[0m\n", addr)
	return srv.ListenAndServe()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func unauthorized(c *gin.Context, mw *ginjwt.GinJWTMiddleware, msg string) {
	c.Header("WWW-Authenticate", "JWT realm="+mw.Realm)
	mw.Unauthorized(c, 400, msg)
	c.Abort()
}

func cors(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", strings.Join([]string{
		"Content-Type",
		"Authorization",
	}, ","))
	c.Next()
}
