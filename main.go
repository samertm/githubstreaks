package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/flosch/pongo2"
	"github.com/google/go-github/github"
	"github.com/gorilla/context"
	"github.com/gorilla/schema"
	"github.com/samertm/githubstreaks/conf"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

var _ = pongo2.Must(pongo2.FromFile("templates/base.html"))

var errorTemplate = pongo2.Must(pongo2.FromFile("templates/error.html"))

type errorTemplateVars struct {
	Message string
	Code    int
}

func RenderTemplate(t *pongo2.Template, w io.Writer, data interface{}) error {
	return t.ExecuteWriter(pongo2.Context{"v": data}, w)
}

var indexTemplate = pongo2.Must(pongo2.FromFile("templates/index.html"))

type indexTemplateVars struct {
	Login  string
	Email  string
	Groups []Group

	NeedEmail bool
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s)
	v := indexTemplateVars{}
	if u != nil {
		v.Login = u.Login
		// Check whether we need to ask for their email.
		if !u.Email.Valid {
			v.NeedEmail = true
		} else {
			v.Email = u.Email.String
		}
		gs, err := GetGroups(*u)
		if err != nil {
			return fmt.Errorf("Error getting groups: %s", err)
		}
		v.Groups = gs
	}
	return RenderTemplate(indexTemplate, w, v)
}

func serveLogin(c web.C, w http.ResponseWriter, r *http.Request) error {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
	return HTTPRedirect{To: url, Code: http.StatusSeeOther}
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
	return HTTPRedirect{To: "/", Code: http.StatusSeeOther}
}

type saveEmailForm struct {
	Email string `schema:"email"`
}

func serveSaveEmail(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s)
	if u == nil {
		return fmt.Errorf("User is not authed")
	}
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("Error parsing form: %s", err)
	}
	var form saveEmailForm
	err := schema.NewDecoder().Decode(&form, r.PostForm)
	if err != nil {
		return fmt.Errorf("Error decoding form: %s", err)
	}
	if err := SetEmail(*u, form.Email); err != nil {
		return fmt.Errorf("Error setting email: %s", err)
	}
	return HTTPRedirect{To: "/", Code: http.StatusSeeOther}
}

// SAMER: Add some auth-detecting middleware?
func serveCreateGroup(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s)
	if u == nil {
		return fmt.Errorf("User is not authed")
	}
	g, err := CreateGroup(*u)
	if err != nil {
		return err
	}
	return HTTPRedirect{To: GroupURL(g), Code: http.StatusSeeOther}
}

var groupTemplate = pongo2.Must(pongo2.FromFile("templates/group.html"))

type groupTemplateVars struct {
	Login   string // SAMER: Add CommonTemplateVars.
	GroupID int
}

// SAMER: The middleware should be storing the user in some struct
// that's being passed forwards...
func serveGroup(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s)
	if u == nil {
		return fmt.Errorf("User is not authed")
	}
	gid, err := strconv.Atoi(c.URLParams["group_id"]) // SAMER: Some type of type checking?
	if err != nil {
		return fmt.Errorf("Error parsing group_id %s: %s", c.URLParams["group_id"], err)
	}
	// SAMER: Check that the user is in the group.
	return RenderTemplate(groupTemplate, w, groupTemplateVars{Login: u.Login, GroupID: gid})
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
	goji.Get("/login", handler(serveLogin))
	goji.Get("/github_callback", handler(serveGitHubCallback))
	goji.Post("/save_email", handler(serveSaveEmail))

	goji.Get("/group/:group_id", handler(serveGroup))
	goji.Post("/create_group", handler(serveCreateGroup))

	goji.Serve()
}
