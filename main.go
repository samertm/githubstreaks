package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/burntsushi/toml"
	"github.com/google/go-github/github"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

var indexTemplate = template.Must(template.ParseFiles("templates/index.html"))

type indexTemplateVars struct {
	Login string
}

var groupTemplate = template.Must(template.ParseFiles("templates/group.html"))

type groupTemplateVars struct {
	GroupID string
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) {
	s := getSession(c)
	u := getUser(s) // SAMER: Change to auth.
	v := indexTemplateVars{}
	if u != nil {
		v.Login = u.Login
	}
	indexTemplate.Execute(w, v)
}

func serveGroup(c web.C, w http.ResponseWriter, r *http.Request) {
	groupID := c.URLParams["group_id"]
	groupTemplate.Execute(w, groupTemplateVars{GroupID: groupID})
}

func serveLogin(c web.C, w http.ResponseWriter, r *http.Request) {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func serveGitHubCallback(c web.C, w http.ResponseWriter, r *http.Request) {
	s := getSession(c)
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		// SAMER: Redirect to error page.
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get("")
	if err != nil {
		fmt.Printf("client.Users.Get() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)
	// SAMER: Save user to DB.
	s.Values[userIDSessionKey] = 1
	if err := s.Save(r, w); err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

var userIDSessionKey = "user_id"

func getUser(s *sessions.Session) *User {
	id, ok := s.Values[userIDSessionKey]
	if !ok {
		return nil
	}
	// SAMER: Put a DB lookup here.
	return &User{ID: id.(int), Login: "blank"}
}

func getSession(c web.C) *sessions.Session {
	return c.Env["session"].(*sessions.Session)
}

type User struct {
	ID    int
	Login string
}

var (
	oauthConf = &oauth2.Config{
		ClientID:     config.GitHubID,
		ClientSecret: config.GitHubSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
	oauthStateString = "somerandomstring"
)

func applySessions(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		c.Env["session"] = session
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

var store = sessions.NewCookieStore(sha256.New().Sum(nil)) // SAMER: Make this secure.

type Config struct {
	GitHubID     string
	GitHubSecret string
}

var config Config

func init() {
	if _, err := toml.DecodeFile("conf.toml", &config); err != nil {
		log.Fatalf("Error decoding conf: %s", err)
	}
}

func main() {
	// Serve static files.
	staticDirs := []string{"bower_components", "res"}
	for _, d := range staticDirs {
		static := web.New()
		pattern, prefix := fmt.Sprintf("/%s/*", d), fmt.Sprintf("/%s/", d)
		static.Get(pattern, http.StripPrefix(prefix, http.FileServer(http.Dir(d))))
		http.Handle(prefix, static)
	}

	goji.Use(applySessions)
	goji.Use(context.ClearHandler)

	goji.Get("/", serveIndex)
	goji.Get("/group/:group_id", serveGroup)
	goji.Get("/login", serveLogin)
	goji.Get("/github_callback", serveGitHubCallback)
	goji.Serve()
}
