package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/google/go-github/github"
	"github.com/samertm/githubstreaks/db"
)

type User struct {
	UID   int    `db:"uid"`
	Login string `db:"login"`
	// SAMER: Make this unique.
	Email                sql.NullString `db:"email"`
	AccessToken          sql.NullString `db:"access_token"`
	ExpiresOn            *time.Time     `db:"expires_on"`
	CommitsLastUpdatedOn *time.Time     `db:"commits_last_updated_on"`
}

var userSchema = `
CREATE TABLE IF NOT EXISTS "user" (
  uid SERIAL PRIMARY KEY,
  login text NOT NULL,
  email text,
  access_token text,
  expires_on timestamp,
  commits_last_updated_on timestamp
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
		return User{}, wrapError(err)
	}
	// Get the user one last time.
	u, err = GetUser(UserSpec{Login: login})
	if err != nil {
		return User{}, wrapError(err)
	}
	return u, nil
}

func CreateUser(login string) error {
	query := `INSERT INTO "user"(login) VALUES ($1)`
	if _, err := db.DB.Exec(query, login); err != nil {
		return wrapError(err)
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

	err := db.DB.Get(&u, fmt.Sprintf(`SELECT * from "user" WHERE %s=$1`, where.col), where.val)
	if err != nil {
		return User{}, wrapError(err)
	}
	return u, nil
}

func GetUserCommits(u User, after time.Time) ([]Commit, error) {
	b := db.Binder{}
	query := `SELECT * FROM commit
WHERE uid = ` + b.Bind(u.UID) + ` AND author_date > ` + b.Bind(after)
	var commits []Commit
	if err := db.DB.Select(&commits, query, b.Items...); err != nil {
		return nil, wrapError(err)
	}
	return commits, nil
}

func SetEmail(u User, email string) error {
	b := &db.Binder{}
	query := `UPDATE "user" SET email = ` + b.Bind(email) + " " +
		"WHERE uid = " + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return wrapError(err)
	}
	return nil
}

func SetAccessToken(u User, token string, expiresIn string) error {
	e, err := strconv.Atoi(expiresIn)
	if err != nil {
		return wrapError(err)
	}
	expiresOn := time.Now().Add(time.Duration(e) * time.Second)
	b := &db.Binder{}
	query := `UPDATE "user" SET access_token = ` + b.Bind(token) + ", " +
		"expires_on = " + b.Bind(expiresOn) + " " +
		"WHERE uid = " + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return wrapError(err)
	}
	return nil
}

func SetCommitsLastUpdatedOn(u User, t time.Time) error {
	b := db.Binder{}
	query := `UPDATE "user" SET commits_last_updated_on = ` + b.Bind(t) + ` ` +
		`WHERE uid = ` + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return wrapError(err)
	}
	return nil
}

// Figure out how far back we need to look to update the user's data.
//
// UpdateTime returns u.CommitsLastUpdatedOn if it is non-nil, else it
// returns the beginning of the day for the user's oldest group. If
// the user does not belong to any groups, it returns the beginning of
// today.
func UpdateTime(u User) (time.Time, error) {
	// SAMER: Rethink this. I may want to update people even if
	// they don't belong to a group.
	if u.CommitsLastUpdatedOn != nil {
		return *u.CommitsLastUpdatedOn, nil
	}
	gs, err := GetGroups(u)
	if err != nil {
		return time.Time{}, wrapError(err)
	}
	if len(gs) == 0 {
		// Return the beginning of today.
		return beginningOfDay(time.Now()), nil
	}
	oldestGroup := gs[0].CreatedOn
	for _, g := range gs {
		if g.CreatedOn.After(oldestGroup) {
			oldestGroup = g.CreatedOn
		}
	}
	return beginningOfDay(beginningOfDay(oldestGroup)), nil
}

func beginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// SAMER: Pick another name.
type Group struct {
	GID       int       `db:"gid"`
	CreatedOn time.Time `db:"created_on"`
	//UIDs []int `db:"uids"`
	// SAMER: Group name?
}

var groupSchema = `
CREATE TABLE IF NOT EXISTS "group" (
  gid SERIAL PRIMARY KEY,
  created_on timestamp NOT NULL
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
  uid integer REFERENCES "user" (uid),
  gid integer REFERENCES "group" (gid)
)`

func init() {
	db.DB.MustExec(userGroupSchema)
}

func CreateGroup(u User) (Group, error) {
	b := &db.Binder{}
	query := `
WITH g AS (
  INSERT INTO "group"(gid, created_on) VALUES (DEFAULT, current_timestamp) RETURNING *
), i AS (
  INSERT INTO user_group(uid, gid)
    SELECT ` + b.Bind(u.UID) + `, gid FROM g
)
SELECT gid FROM g`
	var g Group
	if err := db.DB.Get(&g, query, b.Items...); err != nil {
		return Group{}, wrapErrorf(err, "error creating group for %s", u.Login)
	}
	return g, nil
}

// SAMER: This should be baked into the router. Use gorilla.Mux?
func GroupURL(g Group) string {
	return "/group/" + strconv.Itoa(g.GID)
}

func GetGroup(gid int) (Group, error) {
	b := &db.Binder{}
	query := `SELECT * FROM "group" WHERE gid = ` + b.Bind(gid)
	var g Group
	if err := db.DB.Get(&g, query, b.Items...); err != nil {
		return Group{}, wrapError(err)
	}
	return g, nil
}

func GetGroups(u User) ([]Group, error) {
	// TESTING
	return nil, wrapErrorf(fmt.Errorf("WHOOPS, made a mistake"), "my bad %d", "dog")
	b := &db.Binder{}
	query := `
SELECT * FROM "group"
  WHERE gid IN
    (SELECT gid FROM user_group WHERE uid = ` + b.Bind(u.UID) + `)
ORDER BY gid ASC`
	var gs []Group
	if err := db.DB.Select(&gs, query, b.Items...); err != nil {
		return nil, wrapErrorf(err, "Error retrieving groups for %s (%d): %s", u.Login, u.UID)
	}
	return gs, nil
}

func GetGroupUsers(g Group) ([]User, error) {
	// Too lazy to figure out how joining works.
	b := db.Binder{}
	query := `SELECT uid FROM user_group WHERE gid = ` + b.Bind(g.GID)
	var uids []int
	if err := db.DB.Select(&uids, query, b.Items...); err != nil {
		return nil, wrapError(err)
	}
	us := make([]User, 0, len(uids))
	for _, uid := range uids {
		u, err := GetUser(UserSpec{UID: uid})
		if err != nil {
			return nil, wrapError(err)
		}
		us = append(us, u)
	}
	return us, nil
}

func GetGroupAllCommits(g Group) ([]Commit, error) {
	us, err := GetGroupUsers(g)
	if err != nil {
		return nil, wrapError(err)
	}
	var cs []Commit
	for _, u := range us {
		// SAMER: Clean up 'beginningOfDay' stuff.
		c, err := GetUserCommits(u, beginningOfDay(g.CreatedOn))
		if err != nil {
			return nil, wrapError(err)
		}
		cs = append(cs, c...)
	}
	return cs, nil
}

// SAMER: Make these methods... Or have a consistant naming scheme +
// explain in a doc comment.
func UpdateGroupCommits(g Group) error {
	us, err := GetGroupUsers(g)
	if err != nil {
		return wrapError(err)
	}
	for _, u := range us {
		if err := UpdateUserCommits(u); err != nil {
			return wrapError(err)
		}
	}
	return nil
}

type Commit struct {
	SHA        string    `db:"sha"`
	UID        int       `db:"uid"`
	AuthorDate time.Time `db:"author_date"`
	RepoName   string    `db:"repo_name"`
	Message    string    `db:"message"`
}

// SAMER: repo_name -> full_repo_name?
var commitSchema = `
CREATE TABLE IF NOT EXISTS "commit" (
  sha text PRIMARY KEY,
  uid integer REFERENCES "user" (uid) NOT NULL,
  author_date timestamp NOT NULL,
  repo_name text NOT NULL,
  message text NOT NULL
)`

func init() {
	db.DB.MustExec(commitSchema)
}

type GitHubCommitRepo struct {
	github.Commit
	RepoName string
}

func FetchRecentCommits(u User, until time.Time) ([]GitHubCommitRepo, error) {
	// Figure out ETag stuff.
	// https://developer.github.com/v3/activity/events/
	// SAMER: Should I keep recreating UnauthedGitHubClient?
	client := UnauthedGitHubClient()
	es, _, err := client.Activity.ListEventsPerformedByUser(u.Login, true, nil)
	if err != nil {
		return nil, wrapError(err)
	}
	var cs []GitHubCommitRepo
	for _, e := range es {
		if *e.Type != "PushEvent" {
			continue
		}
		repoUser, repoName := SplitRepoName(*e.Repo.Name)
		PushEventCommits := e.Payload().(*github.PushEvent).Commits
		for _, pec := range PushEventCommits {
			c, _, err := client.Git.GetCommit(repoUser, repoName, *pec.SHA)
			if err != nil {
				return nil, wrapError(err)
			}
			cs = append(cs, GitHubCommitRepo{
				Commit:   *c,
				RepoName: *e.Repo.Name,
			})
		}
	}
	if len(cs) == 0 {
		return nil, nil
	}
	return cs, nil
}

func SplitRepoName(fullRepoName string) (userName, repoName string) {
	s := strings.Split(fullRepoName, "/")
	return s[0], s[1]
}

func UpdateUserCommits(u User) error {
	t, err := UpdateTime(u)
	if err != nil {
		return wrapError(err)
	}
	cs, err := FetchRecentCommits(u, t)
	if err != nil {
		return wrapError(err)
	}
	for _, c := range cs {
		if err := CreateCommit(u, c); err != nil {
			return wrapError(err)
		}
	}
	if err := SetCommitsLastUpdatedOn(u, time.Now()); err != nil {
		return wrapError(err)
	}
	return nil
}

// SAMER: Submit a PR to go-github for Commit.GitHubAuthor.
func CreateCommit(u User, c GitHubCommitRepo) error {
	if c.Message == nil {
		c.Message = github.String("")
	}
	b := db.Binder{}
	query := `
INSERT INTO commit(sha, uid, author_date, repo_name, message)
  VALUES (` + b.Bind(*c.SHA) + `, ` + b.Bind(u.UID) + `, ` + b.Bind(*c.Author.Date) + `, ` +
		b.Bind(c.RepoName) + `, ` + b.Bind(*c.Message) + `)`
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		// Ignore if we've seen this commit.
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return wrapError(err)
	}
	return nil
}

// // SAMER: Handle duplicates?
// func CreateEvent(u User, e github.Event) error {
// 	b := db.Binder{}
// 	query := `
// INSERT INTO "event"(egithub_id, uid)
//   VALUES (` + b.Bind(e.ID) + `, ` + b.Bind(u.UID) + `)`
// 	if _, err := db.DB.Exec(query, b.Items); err != nil {
// 		// Ignore errors on duplicate events.
// 		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
// 			return nil
// 		}
// 		return err
// 	}
// 	return nil
// }
