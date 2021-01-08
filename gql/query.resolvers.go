package gql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/gql/internal/graph"
)

func (r *queryResolver) Courses(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	return resolveCourses(ctx, r.DB, limit, offset, subject)
}

func (r *queryResolver) Blueprints(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.CourseBlueprint, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Catalog(ctx context.Context, limit *int, offset *int, subject *string) ([]*catalog.Course, error) {
	return resolveCatalog(ctx, r.DB, limit, offset, subject)
}

func (r *queryResolver) Course(ctx context.Context, id int) (*catalog.Course, error) {
	var e catalog.Course
	err := r.DB.Get(&e, "select * from	course where id = $1", id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *queryResolver) Subjects(ctx context.Context) ([]*graph.Subject, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns graph.QueryResolver implementation.
func (r *Resolver) Query() graph.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
