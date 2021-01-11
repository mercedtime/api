package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/ulule/limiter/v3"
	ginlimit "github.com/ulule/limiter/v3/drivers/middleware/gin"

	"github.com/mercedtime/api/users"
)

func authorize(r *http.Request, u *users.User) bool {
	path := r.URL.Path
	if r.Method == "POST" || r.Method == "DELETE" {
		if strings.HasSuffix(path, fmt.Sprintf("/user/%d", u.ID)) ||
			strings.HasSuffix(path, fmt.Sprintf("/user/%d/", u.ID)) {
			return true
		} else if strings.HasSuffix(path, "/user") {
			return u.IsAdmin
		}
	}
	switch path {
	case "/admin":
		return u.IsAdmin
	case "/api/v1/unauthorized": // for testing
		return false
	default:
		return r.Method == "GET"
	}
}

// TODO should be protected
// only allow: this user, admin
func (a *App) getUser(c *gin.Context) {
	var (
		id  int
		err error
	)
	rawid, ok := c.Params.Get("id")
	if !ok {
		c.AbortWithStatusJSON(400, &Error{
			Msg:    "no id given",
			Status: 400,
		})
		return
	}

	// If the url parameter was self, get the
	// ID of the current user.
	if rawid == "self" {
		id, err = getSelfID(a.jwtIdentidyKey, c)
		if err != nil {
			err = &Error{Msg: "could not get user:" + err.Error()}
		}
	} else {
		// If not self, parse the url parameter
		id, err = strconv.Atoi(rawid)
		if err != nil {
			err = &Error{Msg: "id is not a number"}
		}
	}
	if err != nil {
		c.AbortWithStatusJSON(400, err)
		return
	}

	u, err := a.GetUser(users.User{ID: id})
	if err != nil {
		senderr(c, users.ErrUserNotFound, 404)
		return
	}
	c.JSON(200, u)
}

func getSelfID(key string, c *gin.Context) (int, error) {
	identity, ok := c.Get(key)
	if !ok {
		return 0, errors.New("no identity")
	}
	if user, ok := identity.(*users.User); ok {
		return user.ID, nil
	}
	return 0, errors.New("could not get identity")
}

// PostUser handles user creation
// TODO: should be protected, only admin
func (a *App) PostUser(c *gin.Context) {
	u, err := a.createUser(c)
	if err != nil {
		log.Printf("%T %v\n", err, err)
	}
	switch e := err.(type) {
	case *pq.Error:
		if e.Code == "23505" {
			c.AbortWithStatusJSON(400, &Error{"Duplicate username or email", 400})
		} else {
			c.AbortWithStatusJSON(500, &Error{e.Detail, 500})
		}
		return
	case *Error:
		c.AbortWithStatusJSON(e.Status, e)
		return
	default:
		break
	}
	switch err {
	case nil:
		c.JSON(201, u)
		return
	case users.ErrInvalidUser:
		c.AbortWithStatusJSON(400, &Error{Msg: "must give a username or email", Status: 400})
	default:
		c.AbortWithStatusJSON(500, gin.H{"error": err})
	}
}

// SilentCreateUser will create a new user without writing to the response body
func (a *App) SilentCreateUser(c *gin.Context) {
	u, err := a.createUser(c)
	switch e := err.(type) {
	case nil:
		c.Set("new-user", u)
		c.Next()
	case *pq.Error:
		if e.Code == "23505" {
			c.AbortWithStatusJSON(400, &Error{"Duplicate username or email", 400})
		} else {
			c.AbortWithStatusJSON(400, &Error{e.Detail, 400})
		}
	case *Error:
		c.AbortWithStatusJSON(e.Status, e)
	default:
		switch err {
		case users.ErrInvalidUser:
			c.AbortWithStatusJSON(400, &Error{Msg: "must give a username or email", Status: 400})
		default:
			c.AbortWithStatusJSON(500, gin.H{"error": err})
		}
	}
}

func (a *App) createUser(c *gin.Context) (*users.User, error) {
	type user struct {
		users.User
		Password string
	}
	u := user{}
	err := c.BindJSON(&u)
	if err != nil {
		return nil, &Error{"could not read request body", 400}
	}
	// TODO check auth for permissions to set is_admin
	u.IsAdmin = false
	u.CreatedAt = time.Time{} // this is taken care of by postgres
	u.ID = 0                  // database handles this
	if u.Password == "" {
		return nil, ErrStatus(400, "no password for new user")
	}
	return a.CreateUser(&u.User, u.Password)
}

func (a *App) deleteUser(c *gin.Context) {
	u := users.User{ID: c.GetInt("id")}
	switch err := users.Delete(a.DB, u); err {
	case nil:
		c.JSON(200, &Msg{
			Msg:    "user successfully deleted",
			Status: 200,
		})
	case users.ErrUserNotFound:
		senderr(c, err, 404)
	default:
		senderr(c, err, 500)
	}
}

func createUserRateLimit(store limiter.Store) gin.HandlerFunc {
	return ginlimit.NewMiddleware(limiter.New(
		store,
		limiter.Rate{
			Period: time.Minute,
			Limit:  5,
		},
	))
}
