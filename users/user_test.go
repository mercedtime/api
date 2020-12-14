package users

import "testing"

func TestCreateUser(t *testing.T) {
	u := User{
		Name:  "test-admin",
		Email: "testadmin@email.com",
	}
	err := u.Save()
	if err == nil {
		t.Fatal("error should be nil for no password hash")
	}
	if err = u.setPassword("password1"); err != nil {
		t.Fatal(err)
	}
	if u.Hash == nil {
		t.Error("no password generated")
	}
	if !u.PasswordOK("password1") {
		t.Error("password hasing failed")
	}
}
