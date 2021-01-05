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
	type user struct {
		users.User
		Password string
	}
	u := user{}
	err := c.BindJSON(&u)
	if err != nil {
		c.JSON(500, NewErr("could not read body"))
		return
	}

	// TODO check auth for permissions to set is_admin
	u.IsAdmin = false
	u.CreatedAt = time.Time{} // this is taken care of by postgres
	u.ID = 0

	if u.Password == "" {
		c.JSON(400, ErrStatus(400, "no password for new user"))
		return
	}
	_, err = a.CreateUser(&u.User, u.Password)
	switch e := err.(type) {
	case nil:
		c.JSON(200, u.User)
	case *pq.Error:
		if e.Code == "23505" {
			c.AbortWithStatusJSON(400, &Error{"Duplicate username or email", 400})
		}
	default:
		senderr(c, err, 500)
	}
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
