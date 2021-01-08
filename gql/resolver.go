package gql

import "github.com/jmoiron/sqlx"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is a graphql query resolver.
type Resolver struct {
	DB *sqlx.DB
}
