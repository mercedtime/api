package db

import "github.com/jmoiron/sqlx"

var db *sqlx.DB = nil

// Set will set the global database connection.
func Set(connection *sqlx.DB) {
	db = connection
}

// Get will get the database
func Get() *sqlx.DB {
	return db
}
