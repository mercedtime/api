package users

import "errors"

var (
	// ErrUserNotFound is returned when the user does not
	// exists or was not found given the search parameters.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidUser is returned when the user given is not valid
	ErrInvalidUser = errors.New("invalid user")
)
