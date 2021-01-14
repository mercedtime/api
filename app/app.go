package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	apidb "github.com/mercedtime/api/db"
	"github.com/mercedtime/api/db/models"
	"github.com/mercedtime/api/users"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	_ "github.com/lib/pq" // app package relies on pq for postgres
)

// App is the main app
type App struct {
	DB        *sqlx.DB
	Config    *Config
	Engine    *gin.Engine
	RateStore limiter.Store
	Protected gin.HandlerFunc

	jwtIdentidyKey string
}

// New creates a new app
func New(conf *Config) (*App, error) {
	db, err := sqlx.Connect(conf.Database.Driver, conf.GetDSN())
	if err != nil {
		return nil, err
	}
	apidb.Set(db)
	a := &App{
		DB:     db,
		Config: conf,
	}
	if conf.InMemoryRateStore {
		a.RateStore = memory.NewStore()
	} else {
		return nil, errors.New("don't know how to create rate limit storage")
	}
	return a, nil
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
	return nil, &Error{"not enough info to find user", 500}
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

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Engine.ServeHTTP(w, r)
}

var _ http.Handler = (*App)(nil)

// Msg is a standardized response message
// for misc json endpoints.
type Msg struct {
	Msg    string `json:"message"`
	Status int    `json:"status,omitempty"`
}

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
			"[\x1b[35m%s\x1b[0m] \"\x1b[34m%s\x1b[0m\" %6v %s%d%s %s %s\n",
			f.TimeStamp.Format(time.Stamp),
			f.ClientIP,
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
		a.jwtIdentidyKey = "identity"
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
			c.AbortWithStatusJSON(code, &Error{
				Status: code,
				Msg:    message,
			})
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			var (
				resp = gin.H{
					"code":   code,
					"token":  token,
					"expire": expire.Format(time.RFC3339),
				}
			)
			if u, ok := c.Get("new-user"); ok {
				resp["code"] = http.StatusCreated
				resp["user"] = u
				c.JSON(http.StatusCreated, resp)
			} else {
				c.JSON(http.StatusOK, resp)
			}
		},
	})
	if err != nil {
		return nil, err
	}
	a.Protected = middleware.MiddlewareFunc()
	return middleware, nil
}

func (a *App) authenticate(c *gin.Context) (interface{}, error) {
	newuser, ok := c.Get("new-user")
	if ok && newuser != nil {
		return newuser, nil
	}
	type login struct {
		Name     string `form:"name" json:"name" binding:"required"`
		Password string `form:"password" json:"password" binding:"required"`
	}
	var l login
	err := c.ShouldBind(&l)
	if err != nil {
		return nil, ginjwt.ErrMissingLoginValues
	}
	u, err := users.GetUserByName(a.DB, l.Name)
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
	return authorize(c.Request, u)
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
