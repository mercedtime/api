package users

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// User is a user model
type User struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
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
	_, err = db.NamedExec(`
	  INSERT INTO
		users (name, email, is_admin, hash)
	  VALUES (:name, :email, :is_admin, :hash)`, u)
	return err
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
	row := db.QueryRowx(query, cond)
	err := row.StructScan(u)
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
	if u.Email == "" {
		_, err = u.db.NamedExec(deleteUserBaseQuery, u)
	} else {
		_, err = u.db.NamedExec(deleteUserWithEmail, u)
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
		  hash = :hash`,
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
