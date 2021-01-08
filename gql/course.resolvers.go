package gql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/gql/internal/graph"
)

func (r *courseResolver) Days(ctx context.Context, obj *catalog.Course) ([]*string, error) {
	return resolveDaysOptional(obj.Days), nil
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
	return resolveDays(obj.Days), nil
}

// Course returns graph.CourseResolver implementation.
func (r *Resolver) Course() graph.CourseResolver { return &courseResolver{r} }

// SubCourse returns graph.SubCourseResolver implementation.
func (r *Resolver) SubCourse() graph.SubCourseResolver { return &subCourseResolver{r} }

type courseResolver struct{ *Resolver }
type subCourseResolver struct{ *Resolver }
