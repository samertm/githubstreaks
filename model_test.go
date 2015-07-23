package main

import (
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
