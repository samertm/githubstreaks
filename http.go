package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/samertm/githubstreaks/conf"
	"github.com/zenazn/goji/web"
)

func absoluteURL(urlFragment string) string {
	return conf.Config.BaseURL + "/" + strings.TrimPrefix(urlFragment, "/")
}

var userIDSessionKey = "user_id"

func getUser(s *sessions.Session) *User {
	id, ok := s.Values[userIDSessionKey]
	if !ok {
		return nil
	}
	u, err := GetUser(UserSpec{ID: id.(int)})
	if err != nil {
		log.Printf("Error getting user (id %d): %s\n", id, err)
		return nil
	}
	return &u
}

func getSession(c web.C) *sessions.Session {
	return c.Env["session"].(*sessions.Session)
}

func applySessions(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		c.Env["session"] = session
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

var store = sessions.NewCookieStore([]byte(conf.Config.SessionKey))

type HTTPError struct {
	Err  error
	Code int
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("Error code: %d, error: %s", e.Code, e.Err)
}

type HTTPRedirect struct {
	To   string
	Code int
}

func (e HTTPRedirect) Error() string {
	return fmt.Sprintf("Redirect code %d to %s", e.Code, e.To)
}

type handler func(web.C, http.ResponseWriter, *http.Request) error

func (h handler) ServeHTTPC(c web.C, w http.ResponseWriter, r *http.Request) {
	var err error

	defer func() {
		if rv := recover(); rv != nil {
			err = errors.New("handler panic")
			logError(c, r, err, rv)
			handleError(w, r, err)
		}
	}()

	err = h(c, w, r)
	if err != nil {
		if e, ok := err.(HTTPRedirect); ok {
			http.Redirect(w, r, e.To, e.Code)
			return
		}
		logError(c, r, err, nil)
		handleError(w, r, err)
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var message string
	var code int
	if e, ok := err.(HTTPError); ok {
		message = e.Err.Error()
		code = e.Code
	} else {
		message = err.Error()
		code = http.StatusInternalServerError
	}
	w.Header().Set("cache-control", "no-cache")
	w.WriteHeader(code)
	errorTemplate.Execute(w, errorTemplateVars{
		Message: message,
		Code:    code,
	})
}

func logError(c web.C, req *http.Request, err error, rv interface{}) {
	if err != nil {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Error serving %s: %s\n",
			req.URL,
			// SAMER: Wait for PR to merge.
			//c.Env[web.MatchKey].(web.Match).Pattern.String(),
			err)
		if rv != nil {
			fmt.Fprintln(&buf, rv)
			buf.Write(debug.Stack())
		}
		log.Print(buf.String())
	}
}
