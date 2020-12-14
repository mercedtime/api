package app

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mercedtime/api/users"
)

// PostUser handles user creation
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
	id, ok := c.Params.Get("id")
	if !ok {
		senderr(c, errors.New("no user id given"), 400)
		return
	}
	if _, err := a.DB.Exec("DELETE FROM users WHERE id = $1", id); err != nil {
		senderr(c, err, 500)
		return
	}
	c.JSON(200, &Error{
		Msg:    "user successfully deleted",
		Status: 200,
	})
}
