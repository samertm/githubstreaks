package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
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
	Email       sql.NullString `db:"email"`
	AccessToken sql.NullString `db:"access_token"`
	ExpiresOn   *time.Time     `db:"expires_on"`
	// SAMER: Commits last updated on is never used?
	CommitsLastUpdatedOn *time.Time     `db:"commits_last_updated_on"`
	ETag                 sql.NullString `db:"etag"`
}

var userSchema = `
CREATE TABLE IF NOT EXISTS "user" (
  uid SERIAL PRIMARY KEY,
  login text NOT NULL,
  email text,
  access_token text,
  expires_on timestamp,
  commits_last_updated_on timestamp,
  etag text
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
	b := &db.Binder{}
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
	b := &db.Binder{}
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
		return BeginningOfDay(time.Now()), nil
	}
	oldestGroup := gs[0].CreatedOn
	for _, g := range gs {
		if g.CreatedOn.After(oldestGroup) {
			oldestGroup = g.CreatedOn
		}
	}
	return BeginningOfDay(oldestGroup), nil
}

func BeginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func SetETag(u User, etag string) error {
	b := &db.Binder{}
	query := `UPDATE "user" SET etag = ` + b.Bind(etag) + ` ` +
		`WHERE uid = ` + b.Bind(u.UID)
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return wrapError(err)
	}
	return nil
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

func GroupAddUser(g Group, u User) error {
	b := &db.Binder{}
	query := `
INSERT INTO user_group(uid, gid) VALUES (` + b.Bind(u.UID, g.GID) + `)`
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		return wrapErrorf(err, "error adding user %d to group %d", u.UID, g.GID)
	}
	return nil
}

// SAMER: This should be baked into the router. Use gorilla.Mux?
func GroupURL(g Group) string {
	return "/group/" + strconv.Itoa(g.GID)
}

func GroupShareURL(g Group) string {
	return "/group/" + strconv.Itoa(g.GID) + "/join?key=" + GroupSecretKey(g)
}

// SAMER: Make secret key even more secret?
func GroupSecretKey(g Group) string {
	m := md5.New()
	m.Write([]byte(strconv.Itoa(g.GID)))
	m.Write([]byte(strconv.FormatInt(g.CreatedOn.Unix(), 10)))
	return hex.EncodeToString(m.Sum(nil))
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
	b := &db.Binder{}
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
		// SAMER: Clean up 'BeginningOfDay' stuff.
		c, err := GetUserCommits(u, BeginningOfDay(g.CreatedOn))
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
	Additions  int       `db:"additions"`
	Deletions  int       `db:"deletions"`
	//Files []CommitFile
}

type CommitFile struct {
	CommitSHA string `db:"commit_sha"`
	Filename  string `db:"filename"`
	// Status is one of "modified", "removed", "added".
	Status    string `db:"status"`
	Additions int    `db:"additions"`
	Deletions int    `db:"deletions"`
	Patch     string `db:"patch"`
}

// SAMER: repo_name -> full_repo_name?
var (
	commitSchema = `
CREATE TABLE IF NOT EXISTS "commit" (
  sha text PRIMARY KEY,
  uid integer REFERENCES "user" (uid) NOT NULL,
  author_date timestamp NOT NULL,
  repo_name text NOT NULL,
  message text NOT NULL,
  additions integer NOT NULL,
  deletions integer NOT NULL
)`

	commitFileSchema = `
CREATE TABLE IF NOT EXISTS "commit_file" (
  commit_sha text REFERENCES "commit" (sha),
  filename text NOT NULL,
  status text NOT NULL,
  additions integer NOT NULL,
  deletions integer NOT NULL,
  patch text NOT NULL
)`
)

func init() {
	db.DB.MustExec(commitSchema)
	db.DB.MustExec(commitFileSchema)
}

func GetCommit(sha string) (Commit, error) {
	b := &db.Binder{}
	query := `SELECT * FROM commit WHERE sha = ` + b.Bind(sha)
	var c Commit
	if err := db.DB.Get(&c, query, b.Items...); err != nil {
		return Commit{}, err
	}
	return c, nil
}

var sqlNotFound = "no rows in result set"

func CommitExists(sha string) (bool, error) {
	if _, err := GetCommit(sha); err != nil {
		if strings.Contains(err.Error(), sqlNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SAMER: Use functions or methods.
func ShortSHA(sha string) string {
	if len(sha) < 7 {
		return sha
	}
	return sha[:8]
}

func CommitMessageTitle(m string) string {
	return strings.Split(m, "\n")[0]
}

type SortableCommits []Commit

func (s SortableCommits) Len() int           { return len(s) }
func (s SortableCommits) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SortableCommits) Less(i, j int) bool { return s[i].AuthorDate.After(s[j].AuthorDate) }

type CommitGroup struct {
	RepoName  string
	Additions int
	Deletions int
	Commits   []Commit
}

type SortableCommitGroups []CommitGroup

func (s SortableCommitGroups) Len() int      { return len(s) }
func (s SortableCommitGroups) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortableCommitGroups) Less(i, j int) bool {
	return s[i].Commits[0].AuthorDate.After(s[j].Commits[0].AuthorDate)
}

// CommitGroups groups commits by repo, sorted by the most recent
// commits.
func CommitGroups(commits []Commit) []CommitGroup {
	// First, sort commits into CommitGroups by repo.
	cgm := make(map[string]CommitGroup)
	for _, c := range commits {
		cg := cgm[c.RepoName]
		// This is essentially a no-op if cg.RepoName is not empty.
		cg.RepoName = c.RepoName
		cg.Additions += c.Additions
		cg.Deletions += c.Deletions
		cg.Commits = append(cg.Commits, c)
		cgm[c.RepoName] = cg
	}
	// Now, we sort each of the commit arrays by time and stuff
	// them into a slice.
	cgs := make([]CommitGroup, 0, len(cgm))
	for _, cg := range cgm {
		sort.Sort(SortableCommits(cg.Commits))
		cgs = append(cgs, cg)
	}
	// Finally, we sort the CommitGroups.
	sort.Sort(SortableCommitGroups(cgs))
	return cgs
}

type DayCommitGroup struct {
	Day       time.Time
	Additions int
	Deletions int
	Commits   []Commit
}

// DayCommitGroups groups commits by day, sorted by the most recent
// day. We do this by sorting all of the commits by time, descending,
// and then adding them to the current DayCommitGroup until the day
// changes.
func DayCommitGroups(commits []Commit) []DayCommitGroup {
	if len(commits) == 0 {
		return nil
	}
	// First, sort the commits by time, descending.
	sort.Sort(SortableCommits(commits))
	updateDCG := func(dcg *DayCommitGroup, c Commit) {
		dcg.Commits = append(dcg.Commits, c)
		// Update additions and deletions.
		dcg.Additions += c.Additions
		dcg.Deletions += c.Deletions
	}
	var dcgs []DayCommitGroup
	// Initialize currentDCG with the first element in commits.
	var currentDCG DayCommitGroup
	currentDCG.Day = BeginningOfDay(commits[0].AuthorDate)
	updateDCG(&currentDCG, commits[0])
	// Loop over the rest of the commits.
	for i := 1; i < len(commits); i++ {
		c := commits[i]
		if b := BeginningOfDay(c.AuthorDate); currentDCG.Day.After(b) {
			// Append old DCG and initialize new DCG.
			dcgs = append(dcgs, currentDCG)
			currentDCG = DayCommitGroup{Day: b}
			updateDCG(&currentDCG, c)
			continue
		}
		// Update currentDCG with current commit.
		updateDCG(&currentDCG, c)
	}
	// Append last DayCommitGroup.
	dcgs = append(dcgs, currentDCG)
	return dcgs
}

type GitHubCommitRepo struct {
	github.RepositoryCommit
	RepoName string
}

func FetchRecentCommits(u User, transport http.RoundTripper) ([]GitHubCommitRepo, error) {
	client := UnauthedGitHubClient(transport)
	es, resp, err := client.Activity.ListEventsPerformedByUser(u.Login, true, nil)
	// If the response was not modified, then there are no new
	// events (es is nil).
	if resp.StatusCode != http.StatusNotModified && err != nil {
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
			// Don't fetch the commit if we already have a
			// copy of it in the database.
			exists, err := CommitExists(*pec.SHA)
			if err != nil {
				return nil, err
			}
			if exists {
				continue
			}
			c, _, err := client.Repositories.GetCommit(repoUser, repoName, *pec.SHA)
			if err != nil {
				return nil, wrapError(err)
			}
			cs = append(cs, GitHubCommitRepo{
				RepositoryCommit: *c,
				RepoName:         *e.Repo.Name,
			})
		}
	}
	return cs, nil
}

func SplitRepoName(fullRepoName string) (userName, repoName string) {
	s := strings.Split(fullRepoName, "/")
	return s[0], s[1]
}

func UpdateUserCommits(u User) error {
	// SAMER: Do I need this?
	// t, err := UpdateTime(u)
	// if err != nil {
	// 	return wrapError(err)
	// }
	t := NewETagTransport(u.ETag.String)
	var functionFinishedSuccessfully bool // Set this before returning success.
	defer func() {
		etag := t.GetNewETag()
		if functionFinishedSuccessfully {
			if err := SetETag(u, etag); err != nil {
				log.Println(err)
			}
		}
	}()
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
	functionFinishedSuccessfully = true
	return nil
}

func CreateCommit(u User, c GitHubCommitRepo) error {
	if u.Login != *c.Author.Login {
		// Ignore the commit if the author does not match u.
		return nil
	}
	if c.Message == nil {
		c.Message = github.String("")
	}
	b := &db.Binder{}
	query := `
INSERT INTO commit(sha, uid, author_date, repo_name, message, additions, deletions)
  VALUES (` +
		b.Bind(*c.SHA, u.UID, *c.Commit.Author.Date,
			c.RepoName, *c.Commit.Message, *c.Stats.Additions, *c.Stats.Deletions) +
		`)`
	if _, err := db.DB.Exec(query, b.Items...); err != nil {
		// Ignore if we've seen this commit.
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return wrapError(err)
	}
	for _, f := range c.Files {
		b := &db.Binder{}
		// For empty files, Patch is nil.
		if f.Patch == nil {
			f.Patch = github.String("")
		}
		query := `
INSERT INTO commit_file(commit_sha, filename, status, additions, deletions, patch)
  VALUES (` +
			b.Bind(*c.SHA, *f.Filename, *f.Status, *f.Additions, *f.Deletions, *f.Patch) +
			`)`
		if _, err := db.DB.Exec(query, b.Items...); err != nil {
			return wrapError(err)
		}
	}
	return nil
}
