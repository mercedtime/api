package gql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/gql/internal/graph"
)

func (r *courseBlueprintResolver) Crns(ctx context.Context, obj *catalog.CourseBlueprint) ([]int, error) {
	return pqArrToIntArr(obj.CRNs), nil
}

func (r *courseBlueprintResolver) Ids(ctx context.Context, obj *catalog.CourseBlueprint) ([]int, error) {
	return pqArrToIntArr(obj.IDs), nil
}

// CourseBlueprint returns graph.CourseBlueprintResolver implementation.
func (r *Resolver) CourseBlueprint() graph.CourseBlueprintResolver {
	return &courseBlueprintResolver{r}
}

type courseBlueprintResolver struct{ *Resolver }
