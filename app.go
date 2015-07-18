package main

import (
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/zenazn/goji/web"
)

type App struct {
	// Session is never nil.
	Session *sessions.Session
	// User will be nil if the user is not authed. Check
	// App.Authed() or use an explicit nil check before
	// dereferencing User.
	User *User
}

func NewApp(c web.C) App {
	s := getSession(c)
	u := getUser(s)
	return App{Session: s, User: u}
}

// Authed returns nil if the user is authed, otherwise returns an
// error. Use this to check that a user is authed. If auth is nil,
// a.User will never be nil.
func (a App) Authed() error {
	if a.User == nil {
		return fmt.Errorf("User is not authed")
	}
	return nil
}
