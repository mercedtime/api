package app

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/mercedtime/api/users"
)

// TODO should be protected
// only allow: this user, admin
func (a *App) getUser(c *gin.Context) {
	u, err := a.GetUser(users.User{ID: c.GetInt("id")})
	if err != nil {
		senderr(c, users.ErrUserNotFound, 404)
		return
	}
	c.JSON(200, u)
}

// PostUser handles user creation
// TODO: should be protected, only admin
func (a *App) PostUser(c *gin.Context) {
	u, err := a.createUser(c)
	switch e := err.(type) {
	case nil:
		c.JSON(201, u)
	case *pq.Error:
		if e.Code == "23505" {
			c.AbortWithStatusJSON(400, &Error{"Duplicate username or email", 400})
		}
	case *Error:
		c.AbortWithStatusJSON(e.Status, e)
	default:
		break
	}

	switch err {
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
