package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/samertm/githubstreaks/db"
)

type User struct {
	UID   int    `db:"uid"`
	Login string `db:"login"`
	// SAMER: Make this unique.
	Email       sql.NullString `db:"email"`
	AccessToken sql.NullString `db:"access_token"`
	ExpiresOn   *time.Time     `db:"expires_on"`
}

var userSchema = `
CREATE TABLE IF NOT EXISTS "user" (
  uid SERIAL PRIMARY KEY,
  login TEXT NOT NULL,
  email TEXT,
  access_token TEXT,
  expires_on TIMESTAMP
)
`

func init() {
	db.DB.MustExec(userSchema)
}

type UserSpec struct {
	UID   int
	Login string
}

func GetCreateUser(login string) (User, error) {
	// Try to get the user once.
	u, err := GetUser(UserSpec{Login: login})
	if err == nil {
		// User exists, return them.
		return u, nil
	}
	// Create the user and then get them.
	if err := CreateUser(login); err != nil {
		return User{}, err
	}
	// Get the user one last time.
	return GetUser(UserSpec{Login: login})
}

func CreateUser(login string) error {
	query := "INSERT INTO person(login) VALUES ($1)"
	if _, err := db.DB.Exec(query, login); err != nil {
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
	if us.UID != 0 {
		where.col = "uid"
		where.val = strconv.Itoa(us.UID)
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

func SetEmail(u User, email string) error {
	b := &db.Binder{}
	query := "UPDATE person SET email = " + b.Bind(email) + " " +
		"WHERE uid = " + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return err
	}
	return nil
}

func SetAccessToken(u User, token string, expiresIn string) error {
	e, err := strconv.Atoi(expiresIn)
	if err != nil {
		return err
	}
	expiresOn := time.Now().Add(time.Duration(e) * time.Second)
	b := &db.Binder{}
	query := "UPDATE person SET access_token = " + b.Bind(token) + ", " +
		"expires_on = " + b.Bind(expiresOn) + " " +
		"WHERE uid = " + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return err
	}
	return nil
}

// SAMER: Pick another name.
type Group struct {
	GID int `db:"gid"`
	//UIDs []int `db:"uids"`
	// SAMER: Group name?
}

var groupSchema = `
CREATE TABLE IF NOT EXISTS "group" (
  gid SERIAL PRIMARY KEY
)`

func init() {
	db.DB.MustExec(groupSchema)
}

type UserGroup struct {
	UID int `db:"uid"`
	GID int `db:"gid"`
}

var userGroupSchema = `
CREATE TABLE IF NOT EXISTS user_group (
  uid INTEGER REFERENCES "user" (uid),
  gid INTEGER REFERENCES "group" (gid)
)`

func init() {
	db.DB.MustExec(userGroupSchema)
}

func CreateGroup(u User) (Group, error) {
	b := &db.Binder{}
	query := `
WITH g AS (
  INSERT INTO "group"(gid) VALUES (DEFAULT) RETURNING *
), i AS (
  INSERT INTO user_group(uid, gid)
    SELECT ` + b.Bind(u.UID) + `, gid FROM g
)
SELECT gid FROM g`
	var g Group
	if err := db.DB.Get(&g, query, b.Items...); err != nil {
		return Group{}, fmt.Errorf("Error creating group for %s: %s", u.Login, err)
	}
	return g, nil
}

// SAMER: This should be baked into the router. Use gorilla.Mux?
func GroupURL(g Group) string {
	return "/group/" + strconv.Itoa(g.GID)
}

func GetGroups(u User) ([]Group, error) {
	b := &db.Binder{}
	query := `
SELECT * FROM "group"
  WHERE gid IN
    (SELECT gid FROM user_group WHERE uid = ` + b.Bind(u.UID) + `)
ORDER BY gid ASC`
	var gs []Group
	if err := db.DB.Select(&gs, query, b.Items...); err != nil {
		return nil, fmt.Errorf("Error retrieving groups for %s (%d): %s", u.Login, u.UID, err)
	}
	return gs, nil
}

type Event struct {
	ID       int `db:"id"`
	GitHubID int `db:"github_id"`
}
