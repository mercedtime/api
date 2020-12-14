package app

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	sql "github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
	"github.com/mercedtime/api/users"
)

// App is the main app
type App struct {
	DB        *sql.DB
	Config    *Config
	Engine    *gin.Engine
	Protected gin.HandlerFunc

	jwtIdentidyKey string
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
	row := a.DB.QueryRowx("SELECT * FROM instructor WHERE id = $1", id)
	if err := row.StructScan(&inst); err != nil {
		return nil, ErrStatus(500, "could not get instructor")
	}
	return &inst, nil
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
			"[%s] %6v %s%d%s %s %s\n",
			f.TimeStamp.Format(time.Stamp),
			f.Latency,
			statusColor(f.StatusCode), f.StatusCode, "\x1b[0m",
			f.Method,
			f.Path,
		)
	},
}
