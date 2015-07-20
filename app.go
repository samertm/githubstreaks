package main

import (
	"net/http"
	"net/url"

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

// Authed returns nil if the user is authed, otherwise returns a
// *HTTPRedirect that brings the user through the login flow. If r is
// non-nil, a *HTTPRedirect error is returned that will bring the user
// through the login flow and return them to the same page. Use this
// to check that a user is authed. If auth is nil, a.User will never
// be nil.
func (a App) Authed(r *http.Request) *HTTPRedirect {
	if a.User == nil {
		var to string
		if r == nil || r.URL == nil {
			to = "/login"
		} else {
			to = "/login?redirect=" + url.QueryEscape(r.URL.RequestURI())
		}
		return &HTTPRedirect{
			To:   to,
			Code: http.StatusSeeOther,
		}
	}
	return nil
}
