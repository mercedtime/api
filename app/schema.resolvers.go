package app

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mercedtime/api/app/internal/gql"
	"github.com/mercedtime/api/catalog"
)

func (r *courseResolver) Days(ctx context.Context, obj *catalog.Course) ([]*string, error) {
	var days = make([]*string, len(obj.Days))
	for i, day := range obj.Days {
		d := string(day)
		days[i] = &d
	}
	return days, nil
}

func (r *courseResolver) UpdatedAt(ctx context.Context, obj *catalog.Course) (*string, error) {
	d := obj.UpdatedAt.String()
	return &d, nil
}

func (r *courseResolver) Subcourses(ctx context.Context, obj *catalog.Course) ([]*catalog.SubCourse, error) {
	var sub = make([]*catalog.SubCourse, len(obj.Subcourses))
	for i, s := range obj.Subcourses {
		sub[i] = &s
	}
	return sub, nil
}

func (r *queryResolver) Courses(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
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
	err := r.DB.Select(&resp, q, args...)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return resp, nil
}

func (r *queryResolver) Catalog(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
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
	err := r.DB.Select(&resp, q, args...)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return resp, nil
}

func (r *queryResolver) Course(ctx context.Context, id int) (*catalog.Course, error) {
	var e catalog.Course
	err := r.DB.Get(&e, "select * from	course where id = $1", id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *queryResolver) Subjects(ctx context.Context) ([]*gql.Subject, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *subCourseResolver) StartTime(ctx context.Context, obj *catalog.SubCourse) (*string, error) {
	var d = obj.StartTime.String()
	return &d, nil
}

func (r *subCourseResolver) EndTime(ctx context.Context, obj *catalog.SubCourse) (*string, error) {
	var d = obj.EndTime.String()
	return &d, nil
}

func (r *subCourseResolver) UpdatedAt(ctx context.Context, obj *catalog.SubCourse) (*string, error) {
	var d = obj.UpdatedAt.String()
	return &d, nil
}

func (r *subCourseResolver) Days(ctx context.Context, obj *catalog.SubCourse) ([]string, error) {
	var days = make([]string, len(obj.Days))
	for i, day := range obj.Days {
		days[i] = string(day)
	}
	return days, nil
}

// Course returns gql.CourseResolver implementation.
func (r *Resolver) Course() gql.CourseResolver { return &courseResolver{r} }

// Query returns gql.QueryResolver implementation.
func (r *Resolver) Query() gql.QueryResolver { return &queryResolver{r} }

// SubCourse returns gql.SubCourseResolver implementation.
func (r *Resolver) SubCourse() gql.SubCourseResolver { return &subCourseResolver{r} }

type courseResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subCourseResolver struct{ *Resolver }
