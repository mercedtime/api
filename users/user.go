package users

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrUserNotFound is returned when the user does not
	// exists or was not found given the search parameters.
	ErrUserNotFound = errors.New("user not found")
)

// User is a user model
type User struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email,omitempty"`
	IsAdmin   bool      `db:"is_admin" json:"is_admin"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Hash      []byte    `db:"hash" json:"-"`
	db        *sqlx.DB  `db:"-" json:"-"`
}

// Create will create a user
func Create(db *sqlx.DB, u *User, pw string) error {
	u.db = db
	err := u.setPassword(pw)
	if err != nil {
		return err
	}
	query, args, err := db.BindNamed(`
	  INSERT INTO
		users (name, email, is_admin, hash)
	  VALUES (:name, :email, :is_admin, :hash)
	  RETURNING *`, u)
	if err != nil {
		return err
	}
	return db.QueryRowx(query, args...).StructScan(u)
}

// Delete a user
func Delete(db *sqlx.DB, u User) error {
	u.db = db
	return u.Delete()
}

// GetUserByID will get a user from the database by id
func GetUserByID(db *sqlx.DB, id interface{}) (*User, error) {
	var query = "SELECT * FROM users WHERE id = $1"
	return getUser(db, query, id)
}

// GetUserByName will query the database for a user with a specific name
func GetUserByName(db *sqlx.DB, name string) (*User, error) {
	var query = "SELECT * FROM users where name = $1"
	return getUser(db, query, name)
}

func getUser(db *sqlx.DB, query string, cond interface{}) (*User, error) {
	u := &User{}
	err := db.QueryRowx(query, cond).StructScan(u)
	if err != nil {
		return nil, err
	}
	u.db = db
	return u, nil
}

var (
	deleteUserBaseQuery = `
	  DELETE FROM users
	  WHERE
	    id = :id AND
	    name = :name`
	deleteUserWithEmail = deleteUserBaseQuery + ` AND email = :email`
)

// Delete will delete the user
func (u *User) Delete() (err error) {
	var (
		query string
		res   sql.Result
	)
	if u.Email == "" {
		if u.Name == "" {
			query = "DELETE FROM users where id = :id"
		} else {
			query = deleteUserBaseQuery
		}
	} else {
		query = deleteUserWithEmail
	}
	res, err = u.db.NamedExec(query, u)
	if err != nil {
		return err
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrUserNotFound
	}
	return err
}

// Save will save the user to the database
func (u *User) Save() error {
	if u.Hash == nil {
		return errors.New("no password")
	}
	if u.Name == "" {
		return errors.New("no username")
	}
	_, err := u.db.NamedExec(`
	  UPDATE users
	    SET
		  name = :name,
		  email = :email,
		  is_admin = :is_admin,
		  hash = :hash
		WHERE id = :id`,
		u,
	)
	return err
}

// PasswordOK check a password string against the stored password hash
// and returns false if the password is incorrect.
func (u *User) PasswordOK(pw string) (ok bool) {
	err := bcrypt.CompareHashAndPassword(u.Hash, []byte(pw))
	return err == nil
}

// SetDB allows callers to set the internal
// database field on the user struct
func (u *User) SetDB(db *sqlx.DB) {
	u.db = db
}

func (u *User) setPassword(pw string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Hash = hash
	return nil
}
