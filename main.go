package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/gorilla/context"
	"github.com/samertm/githubstreaks/conf"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

var errorTemplate = initializeTemplate("templates/error.html")

type errorTemplateVars struct {
	Message string
	Code    int
}

func initializeTemplate(file string) *template.Template {
	return template.Must(template.ParseFiles("templates/layout.html", file))
}

var indexTemplate = initializeTemplate("templates/index.html")

type indexTemplateVars struct {
	Login string
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s) // SAMER: Change to auth.
	v := indexTemplateVars{}
	if u != nil {
		v.Login = u.Login
	}
	return indexTemplate.Execute(w, v)
}

var groupTemplate = initializeTemplate("templates/group.html")

type groupTemplateVars struct {
	GroupID string
}

func serveGroup(c web.C, w http.ResponseWriter, r *http.Request) error {
	groupID := c.URLParams["group_id"]
	return groupTemplate.Execute(w, groupTemplateVars{GroupID: groupID})
}

func serveLogin(c web.C, w http.ResponseWriter, r *http.Request) error {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
	return HTTPRedirect{
		To:   url,
		Code: http.StatusTemporaryRedirect,
	}
}

func serveGitHubCallback(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	state := r.FormValue("state")
	if state != oauthStateString {
		return fmt.Errorf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fmt.Errorf("oauthConf.Exchange() failed with '%s'\n", err)
	}

	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	ghUser, _, err := client.Users.Get("")
	if err != nil {
		return fmt.Errorf("client.Users.Get() failed with '%s'\n", err)
	}
	log.Printf("Logged in as GitHub user: %s\n", *ghUser.Login)
	// Save user to DB.
	user, err := GetCreateUser(*ghUser.Login)
	if err != nil {
		return fmt.Errorf("Error saving user to the database: %s", err)
	}
	s.Values[userIDSessionKey] = user.ID
	if err := s.Save(r, w); err != nil {
		log.Println(err)
	}
	return HTTPRedirect{
		To:   "/",
		Code: http.StatusTemporaryRedirect,
	}
}

var (
	oauthConf = &oauth2.Config{
		ClientID:     conf.Config.GitHubID,
		ClientSecret: conf.Config.GitHubSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
	oauthStateString = conf.Config.OAuthStateString
)

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

	goji.Get("/", handler(serveIndex))
	goji.Get("/group/:group_id", handler(serveGroup))
	goji.Get("/login", handler(serveLogin))
	goji.Get("/github_callback", handler(serveGitHubCallback))
	goji.Serve()
}
