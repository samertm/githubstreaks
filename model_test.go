package main

import (
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-github/github"
	"github.com/samertm/githubstreaks/db"
)

func TestDayCommitGroups(t *testing.T) {
	days := []time.Time{}
	days = append(days, time.Date(2014, 3, 5, 0, 0, 0, 0, time.UTC))
	days = append(days, time.Date(2014, 3, 3, 0, 0, 0, 0, time.UTC))
	days = append(days, time.Date(2014, 2, 15, 0, 0, 0, 0, time.UTC))
	commits := []Commit{{
		SHA:        "1",
		AuthorDate: time.Date(days[0].Year(), days[0].Month(), days[0].Day(), 16, 15, 15, 15, time.UTC),
	}, {
		SHA:        "2",
		AuthorDate: time.Date(days[0].Year(), days[0].Month(), days[0].Day(), 15, 15, 15, 15, time.UTC),
	}, {
		SHA:        "0",
		AuthorDate: time.Date(days[0].Year(), days[0].Month(), days[0].Day(), 17, 15, 15, 15, time.UTC),
	}, {
		SHA:        "4",
		AuthorDate: time.Date(days[1].Year(), days[1].Month(), days[1].Day(), 15, 15, 15, 15, time.UTC),
	}, {
		SHA:        "3",
		AuthorDate: time.Date(days[1].Year(), days[1].Month(), days[1].Day(), 17, 15, 15, 15, time.UTC),
	}, {
		SHA:        "5",
		AuthorDate: time.Date(days[2].Year(), days[2].Month(), days[2].Day(), 15, 15, 15, 15, time.UTC),
	}}
	dcgs := DayCommitGroups(commits)
	if want := 3; len(dcgs) != want {
		t.Errorf("Got %d dcgs, wanted %d", dcgs, want)
	}
	var counter int
	for _, dcg := range dcgs {
		for _, c := range dcg.Commits {
			if want := BeginningOfDay(c.AuthorDate); dcg.Day != want {
				t.Errorf("Got day %s, wanted %s", dcg.Day, want)
			}
			if want := strconv.Itoa(counter); c.SHA != want {
				t.Errorf("Got SHA %s, wanted %s", c.SHA, want)
			}
			counter++
		}
	}
}

func TestCreateUser(t *testing.T) {
	mdb := db.GetSetMock()
	login := "strange-login"
	sqlmock.ExpectExec(`INSERT INTO "user".*`).
		WithArgs(login).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row.
	if err := CreateUser(login); err != nil {
		t.Error(err)
	}
	if err := mdb.Close(); err != nil {
		t.Error(err)
	}
}

func TestGetCreateUser(t *testing.T) {
	mdb := db.GetSetMock()
	login := "strange-login"
	sqlmock.ExpectQuery(`SELECT \* from "user" WHERE login.*`).
		WithArgs(login).
		WillReturnError(fmt.Errorf("user not found"))
	sqlmock.ExpectExec(`INSERT INTO "user".*`).
		WithArgs(login).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row.
	sqlmock.ExpectQuery(`SELECT \* from "user" WHERE login.*`).
		WithArgs(login).
		WillReturnRows(
		sqlmock.NewRows([]string{"uid", "login"}).
			AddRow(1, login))
	u, err := GetCreateUser(login)
	if err != nil {
		t.Error(err)
	}
	if u.Login != login {
		t.Errorf("Got user login %s, wanted %s", u.Login, login)
	}
	if err := mdb.Close(); err != nil {
		t.Error(err)
	}
}

func TestSetEmail(t *testing.T) {
	mdb := db.GetSetMock()
	email := "something@something.com"
	uid := 1
	sqlmock.ExpectExec(`UPDATE "user" SET email = .* WHERE uid = .*`).
		WithArgs(email, uid).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row.
	if err := SetEmail(User{UID: 1}, email); err != nil {
		t.Error(err)
	}
	if err := mdb.Close(); err != nil {
		t.Error(err)
	}
}

func GetGitHubCommitRepoForTest(login string) GitHubCommitRepo {
	d := time.Now()
	return GitHubCommitRepo{
		RepoName: "someuser/somerepo",
		RepositoryCommit: github.RepositoryCommit{
			SHA: github.String("ffffffffffffffffffffffffffffffffffffffff"),
			Author: &github.User{
				Login: github.String(login),
			},
			Commit: &github.Commit{
				SHA:     github.String("ffffffffffffffffffffffffffffffffffffffff"),
				Author:  &github.CommitAuthor{Date: &d},
				Message: github.String("Some commit message."),
			},
			Stats: &github.CommitStats{
				Additions: github.Int(5),
				Deletions: github.Int(5),
				Total:     github.Int(10),
			},
			Files: []github.CommitFile{{
				SHA:       github.String("rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr"),
				Filename:  github.String("some file name"),
				Additions: github.Int(5),
				Deletions: github.Int(5),
				Changes:   github.Int(10),
				Status:    github.String("modified"),
				Patch:     github.String("Some patch"),
			}},
		},
	}
}

func TestCreateCommit(t *testing.T) {
	mdb := db.GetSetMock()
	u := User{UID: 1, Login: "strange-login"}
	c := GetGitHubCommitRepoForTest(u.Login)
	sqlmock.ExpectBegin()
	sqlmock.ExpectExec("INSERT INTO commit.*").
		WithArgs(*c.SHA, u.UID, *c.Commit.Author.Date, c.RepoName,
		*c.Commit.Message, *c.Stats.Additions, *c.Stats.Deletions).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row.
	f := c.Files[0]
	sqlmock.ExpectExec("INSERT INTO commit_file.*").
		WithArgs(*c.SHA, *f.Filename, *f.Status, *f.Additions, *f.Deletions, *f.Patch).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row.
	sqlmock.ExpectCommit()
	if err := CreateCommit(u, c); err != nil {
		t.Error(err)
	}
	if err := mdb.Close(); err != nil {
		t.Error(err)
	}
}

// Manually test CreateGroup.
func TestCreateGroup(t *testing.T) {
	t.SkipNow()
	b := &db.Binder{}
	query := `
WITH g AS (
  INSERT INTO "group"(gid) VALUES (DEFAULT) RETURNING *
), i AS (
  INSERT INTO user_group(uid, gid)
    SELECT ` + b.Bind(1) + `, gid FROM g
)
SELECT gid FROM g`
	var g Group
	err := db.DB.Get(&g, query, b.Items...)
	log.Println(g, err)
}

// Manually test UserUpdateCommits.
func TestUserUpdateCommits(t *testing.T) {
	t.SkipNow()
	u, err := GetUser(UserSpec{Login: "samertm"})
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdateUserCommits(u); err != nil {
		t.Fatal(err)
	}
}

func TestGetGroupUsers(t *testing.T) {
	t.SkipNow()
	g, err := GetGroup(1)
	if err != nil {
		t.Fatal(err)
	}
	us, err := GetGroupUsers(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range us {
		t.Log(u.Login)
	}
}

func TestGetGroupAllCommits(t *testing.T) {
	t.SkipNow()
	g, err := GetGroup(1)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := GetGroupAllCommits(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		t.Log(c.SHA)
	}
}

func TestGetCommitFailure(t *testing.T) {
	t.SkipNow()
	t.Log(GetCommit("JLFKDJSKLJFLDSK"))
}
