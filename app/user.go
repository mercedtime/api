package app

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mercedtime/api/users"
)

// TODO should be protected
// only allow: this user, admin
func (a *App) getUser(c *gin.Context) {
	u, err := a.GetUser(users.User{ID: c.GetInt("id")})
	if err != nil {
		c.JSON(404, &Error{Msg: "could not find user"})
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
	u.CreatedAt = time.Time{} // zero out the time
	u.ID = 0

	if u.Password == "" {
		c.JSON(400, ErrStatus(400, "no password for new user"))
		return
	}
	if _, err = a.CreateUser(&u.User, u.Password); err != nil {
		senderr(c, err, 500)
		return
	}
	c.JSON(200, u.User)
}

func (a *App) deleteUser(c *gin.Context) {
	if _, err := a.DB.Exec(
		"DELETE FROM users WHERE id = $1", c.GetInt("id")); err != nil {
		senderr(c, err, 500)
		return
	}
	c.JSON(200, &Error{
		Msg:    "user successfully deleted",
		Status: 200,
	})
}
