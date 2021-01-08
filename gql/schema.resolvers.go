package gql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/mercedtime/api/catalog"
	graph1 "github.com/mercedtime/api/gql/internal/graph"
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

func (r *courseBlueprintResolver) Crns(ctx context.Context, obj *catalog.CourseBlueprint) ([]int, error) {
	return pqArrToIntArr(obj.CRNs), nil
}

func (r *courseBlueprintResolver) Ids(ctx context.Context, obj *catalog.CourseBlueprint) ([]int, error) {
	return pqArrToIntArr(obj.IDs), nil
}

func (r *queryResolver) Courses(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	return resolveCourses(r.DB, limit, offset, subject)
}

func (r *queryResolver) Catalog(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	return resolveCatalog(r.DB, limit, offset, subject)
}

func (r *queryResolver) Course(ctx context.Context, id int) (*catalog.Course, error) {
	var e catalog.Course
	err := r.DB.Get(&e, "select * from	course where id = $1", id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *queryResolver) Subjects(ctx context.Context) ([]*graph1.Subject, error) {
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
	return resolveDays(obj.Days), nil
}

// Course returns graph1.CourseResolver implementation.
func (r *Resolver) Course() graph1.CourseResolver { return &courseResolver{r} }

// CourseBlueprint returns graph1.CourseBlueprintResolver implementation.
func (r *Resolver) CourseBlueprint() graph1.CourseBlueprintResolver {
	return &courseBlueprintResolver{r}
}

// Query returns graph1.QueryResolver implementation.
func (r *Resolver) Query() graph1.QueryResolver { return &queryResolver{r} }

// SubCourse returns graph1.SubCourseResolver implementation.
func (r *Resolver) SubCourse() graph1.SubCourseResolver { return &subCourseResolver{r} }

type courseResolver struct{ *Resolver }
type courseBlueprintResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subCourseResolver struct{ *Resolver }
