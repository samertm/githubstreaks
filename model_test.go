package main

import (
	"log"
	"testing"

	"github.com/samertm/syncfbevents/db"
)

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
