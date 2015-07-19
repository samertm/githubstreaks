package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/gorilla/sessions"
	"github.com/samertm/githubstreaks/conf"
	"github.com/zenazn/goji/web"
)

func getParamInt(c web.C, param string) (int, error) {
	v, ok := c.URLParams[param]
	if !ok {
		return 0, errors.Errorf("URLParam %s does not exist in route.", param)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.Errorf("error parsing URLParam %s: %s", v, err)
	}
	return i, nil
}

func absoluteURL(urlFragment string) string {
	return conf.Config.BaseURL + "/" + strings.TrimPrefix(urlFragment, "/")
}

var UIDSessionKey = "user_id"

func getUser(s *sessions.Session) *User {
	uid, ok := s.Values[UIDSessionKey]
	if !ok {
		return nil
	}
	u, err := GetUser(UserSpec{UID: uid.(int)})
	if err != nil {
		log.Printf("Error getting user (uid %d): %s\n", uid, err)
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

// For logError.
var handlerServeFnName = "handler.ServeHTTPC"

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
	RenderTemplate(errorTemplate, w, errorTemplateVars{Message: message, Code: code})
}

func logError(c web.C, req *http.Request, err error, rv interface{}) {
	if err != nil {
		buf := &bytes.Buffer{}
		fmt.Fprintf(buf, "Error serving %s: ", req.URL)
		switch e := err.(type) {
		case *errors.Error:
			fmt.Fprint(buf, "\n"+formatStackFrames(e.StackFrames(), handlerServeFnName))
		default:
			fmt.Fprint(buf, e.Error()+"\n")
		}
		if rv != nil {
			fmt.Fprintln(buf, rv)
			buf.Write(debug.Stack())
		}
		log.Print(buf.String())
	}
}
