package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/samertm/githubstreaks/db"
)

type User struct {
	ID    int
	Login string
	//GitHubID int
}

var userSchema = `
CREATE TABLE IF NOT EXISTS person (
  id SERIAL PRIMARY KEY,
  login TEXT UNIQUE
)
`

func init() {
	db.DB.MustExec(userSchema)
}

type UserSpec struct {
	ID    int
	Login string
}

func GetOrCreateUser(ghUser *github.User) (User, error) {
	// Try to get the user once.
	u, err := GetUser(UserSpec{Login: *ghUser.Login})
	if err == nil {
		// User exists, return them.
		return u, nil
	}
	// Create the user and then get them.
	if err := CreateUser(ghUser); err != nil {
		return User{}, err
	}
	// Get the user one last time.
	return GetUser(UserSpec{Login: *ghUser.Login})
}

func CreateUser(ghUser *github.User) error {
	// SAMER: Bring out ghuser -> user map?
	query := "INSERT INTO person VALUES ($1)"
	if _, err := db.DB.Exec(query, *ghUser.Login); err != nil {
		return err
	}
	return nil
}

func GetUser(us UserSpec) (User, error) {
	u := User{}
	where := struct {
		col string
		val string
	}{}
	if us.ID != 0 {
		where.col = "id"
		where.val = strconv.Itoa(us.ID)
	} else if us.Login != "" {
		where.col = "login"
		where.val = us.Login
	} else {
		return User{}, errors.New("Empty user spec")
	}

	err := db.DB.Get(&u, fmt.Sprintf("SELECT * from person where %s=$1", where.col), where.val)
	if err != nil {
		return User{}, err
	}
	return u, nil
}
