package app

import (
	"log"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/mercedtime/api/users"
)

// NewJWTAuth creates the default jwt auth middleware
func (a *App) NewJWTAuth() (*ginjwt.GinJWTMiddleware, error) {
	if a.jwtIdentidyKey == "" {
		a.jwtIdentidyKey = "id"
	}
	middleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		IdentityKey: a.jwtIdentidyKey,
		Key:         a.Config.Secret,
		// TODO use better a timeout
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 12,

		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		SendCookie:    true,

		Authenticator:   a.authenticate,
		PayloadFunc:     a.jwtPayload,
		Authorizator:    a.authorize,
		IdentityHandler: a.identityHandler,
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, &Error{
				Status: code,
				Msg:    message,
			})
		},
	})
	if err != nil {
		return nil, err
	}
	return middleware, nil
}

func (a *App) authenticate(c *gin.Context) (interface{}, error) {
	type login struct {
		Username string `form:"username" json:"username" binding:"required"`
		Password string `form:"password" json:"password" binding:"required"`
	}
	var l login
	err := c.ShouldBind(&l)
	if err != nil {
		return nil, ginjwt.ErrMissingLoginValues
	}
	u, err := users.GetUserByName(a.DB, l.Username)
	if err != nil {
		return nil, ginjwt.ErrFailedAuthentication
	}
	if u.PasswordOK(l.Password) {
		return u, nil
	}
	return nil, ginjwt.ErrFailedAuthentication
}

func (a *App) authorize(data interface{}, c *gin.Context) bool {
	u, ok := data.(*users.User)
	if !ok {
		return false
	}
	switch c.Request.URL.Path {
	case "/admin":
		return u.IsAdmin
	default:
		return c.Request.Method == "GET"
	}
}

func (a *App) jwtPayload(data interface{}) ginjwt.MapClaims {
	u, ok := data.(*users.User)
	if !ok {
		return ginjwt.MapClaims{}
	}
	return ginjwt.MapClaims{
		a.jwtIdentidyKey: u.ID,
		"name":           u.Name,
		"email":          u.Email,
		"is_admin":       u.IsAdmin,
	}
}

func (a *App) identityHandler(c *gin.Context) interface{} {
	var (
		name   string
		admin  bool
		claims = ginjwt.ExtractClaims(c)
	)
	val, ok := claims["name"]
	if ok {
		name = val.(string)
	}
	val, ok = claims["is_admin"]
	if ok {
		admin = val.(bool)
	}
	id, ok := claims[a.jwtIdentidyKey]
	if !ok {
		log.Println("claims should have the identity key")
		return nil // should not happen
	}
	return &users.User{
		ID:      int(id.(float64)),
		Name:    name,
		IsAdmin: admin,
	}
}
