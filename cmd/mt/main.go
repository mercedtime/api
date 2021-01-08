package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/harrybrwn/config"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/gql"
	"github.com/mercedtime/api/users"
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

	cors := func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", strings.Join([]string{
			"Content-Type",
			"Authorization",
		}, ","))
		c.Next()
	}
	v1 := r.Group("/api/v1")
	if config.GetString("mode") == "debug" || true {
		r.Use(cors)
		v1.Use(cors)
	}
	a.RegisterRoutes(v1)

	r.POST("/graphql", gql.Handler(a.DB))
	r.GET("/graphql/playground", gql.Playground("/graphql"))
	// r.POST("/graphql", a.GraphQLHander())
	// r.GET("/graphql/playground", a.GraphQLPlayground("/graphql"))

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

func login(a *app.App, mw *ginjwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get("new-user")
		if !ok {
			unauthorized(c, mw, mw.HTTPStatusMessageFunc(ginjwt.ErrMissingAuthenticatorFunc, c))
			return
		}
		u := raw.(*users.User)
		if u.Hash == nil {
			unauthorized(c, mw, mw.HTTPStatusMessageFunc(ginjwt.ErrMissingAuthenticatorFunc, c))
			return
		}

		token := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
		claims := token.Claims.(jwt.MapClaims)
		if mw.PayloadFunc != nil {
			for key, value := range mw.PayloadFunc(u) {
				claims[key] = value
			}
		}
		expire := mw.TimeFunc().Add(mw.Timeout)
		claims["exp"] = expire.Unix()
		claims["orig_iat"] = mw.TimeFunc().Unix()
		tokenString, err := token.SignedString(mw.Key)
		if err != nil {
			unauthorized(c, mw, mw.HTTPStatusMessageFunc(ginjwt.ErrFailedTokenCreation, c))
			return
		}
		// set cookie
		if mw.SendCookie {
			expireCookie := mw.TimeFunc().Add(mw.CookieMaxAge)
			maxage := int(expireCookie.Unix() - mw.TimeFunc().Unix())
			if mw.CookieSameSite != 0 {
				c.SetSameSite(mw.CookieSameSite)
			}
			c.SetCookie(
				mw.CookieName,
				tokenString,
				maxage,
				"/",
				mw.CookieDomain,
				mw.SecureCookie,
				mw.CookieHTTPOnly,
			)
		}

		mw.LoginResponse(c, http.StatusOK, tokenString, expire)
		// c.JSON(http.StatusOK, gin.H{
		// 	"user": u,
		// 	"jwt": gin.H{
		// 		"code":   http.StatusOK,
		// 		"token":  token,
		// 		"expire": expire.Format(time.RFC3339),
		// 	},
		// })
	}
}

func unauthorized(c *gin.Context, mw *ginjwt.GinJWTMiddleware, msg string) {
	c.Header("WWW-Authenticate", "JWT realm="+mw.Realm)
	mw.Unauthorized(c, 400, msg)
	c.Abort()
}
