package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

func instructorFromID(app *App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no instructor id"})
			return
		}
		in, err := app.GetInstructor(id)
		if err != nil {
			senderr(c, err, 404)
			return
		}
		if in == nil {
			c.JSON(404, ErrStatus(404, "could not find instructor"))
		}
		c.JSON(200, in)
	}
}

// TODO figure this out, should not only query lectures
// because there are TAs also.
func instructorCourses(db *sqlx.DB) gin.HandlerFunc {
	var query = `
	  SELECT * FROM
	  	lectures
	  WHERE
	    instructor_id = $1`
	return func(c *gin.Context) {
		var list []models.Lecture
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, &Error{
				Msg:    "no id",
				Status: http.StatusBadRequest,
			})
			return
		}
		if err := db.Select(&list, query, id); err != nil {
			c.JSON(500, NewErr(err.Error()))
		}
		if len(list) == 0 {
			c.JSON(404, ErrStatus(404, "could not find resources"))
			return
		}
		c.JSON(200, list)
	}
}
