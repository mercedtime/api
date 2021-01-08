package gql

import (
	"fmt"
	"log"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/gql/internal/graph"
)

//go:generate go run github.com/99designs/gqlgen

// Handler returns a graphql handler function
func Handler(db *sqlx.DB) gin.HandlerFunc {
	h := handler.NewDefaultServer(graph.NewExecutableSchema(
		graph.Config{Resolvers: &Resolver{db}},
	))
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Playground returns a hander func for the graphql playground
func Playground(endpoint string) gin.HandlerFunc {
	h := playground.Handler("GraphQL", endpoint)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func resolveDays(days catalog.Weekdays) []string {
	var res = make([]string, len(days))
	for i, day := range days {
		res[i] = string(day)
	}
	return res
}

func resolveDaysOptional(days catalog.Weekdays) []*string {
	var res = make([]*string, len(days))
	for i, day := range days {
		d := string(day)
		res[i] = &d
	}
	return res
}

func pqArrToIntArr(a pq.Int32Array) []int {
	res := make([]int, len(a))
	for i, v := range a {
		res[i] = int(v)
	}
	return res
}

func resolveCourses(db *sqlx.DB, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	var (
		resp = make([]*catalog.Course, 0, 500)
		q    = `SELECT * FROM course`
		c    = 1
		args = make([]interface{}, 0, 2)
	)
	if subject != nil {
		q += fmt.Sprintf(" WHERE subject = $%d", c)
		c++
		args = append(args, *subject)
	}
	if limit != nil {
		q += fmt.Sprintf(" LIMIT $%d", c)
		c++
		args = append(args, *limit)
	}
	if offset != nil {
		q += fmt.Sprintf(" OFFSET $%d", c)
		c++
		args = append(args, *offset)
	}
	err := db.Select(&resp, q, args...)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return resp, nil
}

func resolveCatalog(db *sqlx.DB, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	var (
		resp = make(catalog.Catalog, 0, 500)
		q    = `SELECT * FROM catalog where type in ('LECT','SEM','STDO')`
		c    = 1
		args = make([]interface{}, 0, 2)
	)
	if subject != nil {
		q += fmt.Sprintf(" AND subject = $%d", c)
		c++
		args = append(args, strings.ToUpper(*subject))
	}
	if limit != nil {
		q += fmt.Sprintf(" LIMIT $%d", c)
		c++
		args = append(args, *limit)
	}
	if offset != nil {
		q += fmt.Sprintf(" OFFSET $%d", c)
		c++
		args = append(args, *offset)
	}
	log.Println(q)
	err := db.Select(&resp, q, args...)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return resp, nil
}
