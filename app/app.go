package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/app/internal/gql"
	"github.com/mercedtime/api/db/models"
	"github.com/mercedtime/api/users"

	_ "github.com/lib/pq" // app package relies on pq for postgres
)

//go:generate go run github.com/99designs/gqlgen

// App is the main app
type App struct {
	DB        *sqlx.DB
	Config    *Config
	Engine    *gin.Engine
	Protected gin.HandlerFunc

	jwtIdentidyKey string
}

// Close the application resourses
func (a *App) Close() error {
	return a.DB.Close()
}

// CreateUser stores a user in the database and sets its private variables
func (a *App) CreateUser(u *users.User, password string) (*users.User, error) {
	return u, users.Create(a.DB, u, password)
}

// GetUser will find a full initialized user give a partially
// initialized user.
func (a *App) GetUser(u users.User) (*users.User, error) {
	if u.ID != 0 {
		return users.GetUserByID(a.DB, u.ID)
	} else if u.Name != "" {
		return users.GetUserByName(a.DB, u.Name)
	}
	return nil, errors.New("not enough info to find user")
}

// GetInstructor will get an instructor by id
func (a *App) GetInstructor(id interface{}) (*models.Instructor, error) {
	var inst models.Instructor
	row := a.DB.QueryRowx(
		"SELECT * FROM instructor WHERE id = $1", id)
	if err := row.StructScan(&inst); err != nil {
		return nil, ErrStatus(500, "could not get instructor")
	}
	return &inst, nil
}

// GraphQLHander returns a graphql hander function
func (a *App) GraphQLHander() func(c *gin.Context) {
	h := handler.NewDefaultServer(gql.NewExecutableSchema(
		gql.Config{Resolvers: &Resolver{a}},
	))
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// GraphQLPlayground returns a hander func for the graphql playground
func (a *App) GraphQLPlayground(endpoint string) gin.HandlerFunc {
	h := playground.Handler("GraphQL", endpoint)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Engine.ServeHTTP(w, r)
}

var _ http.Handler = (*App)(nil)

// Error is an app spesific error
type Error struct {
	Msg    string `json:"error"`
	Status int    `json:"status,omitempty"`
}

// NewErr creates a new error type
func NewErr(msg string) error {
	return &Error{
		Msg:    msg,
		Status: 500,
	}
}

// ErrStatus creates a new error type with a spesific status code
func ErrStatus(status int, msg string) error {
	return &Error{
		Msg:    msg,
		Status: status,
	}
}

func (e *Error) Error() string {
	return e.Msg
}

// LoggerConfig is a config for gin loggers that has cleaner output
var LoggerConfig = gin.LoggerConfig{
	Formatter: func(f gin.LogFormatterParams) string {
		return fmt.Sprintf(
			"[\x1b[35m%s\x1b[0m] %6v %s%d%s %s %s\n",
			f.TimeStamp.Format(time.Stamp),
			f.Latency,
			statusColor(f.StatusCode), f.StatusCode, "\x1b[0m",
			f.Method,
			f.Path,
		)
	},
}

// NewJWTAuth creates the default jwt auth middleware
func (a *App) NewJWTAuth() (*ginjwt.GinJWTMiddleware, error) {
	if a.jwtIdentidyKey == "" {
		a.jwtIdentidyKey = "id"
	}
	middleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		IdentityKey: a.jwtIdentidyKey,
		Key:         []byte(a.Config.Secret),
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
