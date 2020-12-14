package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

func instructorFromID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no instructor id"})
			return
		}
		var inst models.Instructor
		row := db.QueryRow(
			"SELECT id, name FROM instructor WHERE id = $1",
			id,
		)
		if err := row.Scan(&inst.ID, &inst.Name); err != nil {
			c.JSON(500, NewErr(err.Error()))
			return
		}
		c.JSON(200, &inst)
	}
}

func instructorCourses(db *sqlx.DB) gin.HandlerFunc {
	var query = `
	  SELECT * FROM lectures
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
		c.JSON(200, list)
	}
}
