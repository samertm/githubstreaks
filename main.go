package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flosch/pongo2"
	"github.com/go-errors/errors"
	"github.com/google/go-github/github"
	"github.com/gorilla/context"
	"github.com/samertm/githubstreaks/conf"
	"github.com/samertm/githubstreaks/debug"
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

// baseContext is the context that is fed into every template. Keep in
// alphabetical order: abcdefghijklmnopqrstuvwxyz.
var baseContext = pongo2.Context{
	"AbsoluteURL":        AbsoluteURL,
	"CommitGroups":       CommitGroups,
	"CommitMessageTitle": CommitMessageTitle,
	"GetGroupUsers": func(g Group) []User {
		us, _ := GetGroupUsers(g)
		return us
	},
	"GetUser": func(uid int) User {
		u, _ := GetUser(UserSpec{UID: uid})
		return u
	},
	"GroupShareURL": GroupShareURL,
	"GroupURL":      GroupURL,
	"ShortSHA":      ShortSHA,
}

func RenderTemplate(t *pongo2.Template, w io.Writer, data interface{}) error {
	return t.ExecuteWriter(pongo2.Context{"v": data}.Update(baseContext), w)
}

var indexTemplate = pongo2.Must(pongo2.FromFile("templates/index.html"))

type indexTemplateVars struct {
	Login  string
	Email  string
	Groups []Group

	NeedEmail bool
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	v := indexTemplateVars{}
	if a.User != nil {
		v.Login = a.User.Login
		// Check whether we need to ask for their email.
		if !a.User.Email.Valid {
			v.NeedEmail = true
		} else {
			v.Email = a.User.Email.String
		}
		gs, err := GetGroups(*a.User)
		if err != nil {
			return wrapErrorf(err, "error getting groups for User %d", a.User.UID)
		}
		v.Groups = gs
	}
	return RenderTemplate(indexTemplate, w, v)
}

type redirectQuery struct {
	Redirect string `schema:"redirect"`
}

func serveLogin(c web.C, w http.ResponseWriter, r *http.Request) error {
	var q redirectQuery
	if err := SchemaDecoder.Decode(&q, r.URL.Query()); err != nil {
		return wrapError(err)
	}
	oauth := oauthConf // Copy oauthConf so we don't modify it.
	if q.Redirect != "" {
		oauth.RedirectURL = AbsoluteURL("/github_callback?redirect=" + q.Redirect)
	}
	u := oauth.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
	return &HTTPRedirect{To: u, Code: http.StatusSeeOther}
}

func serveGitHubCallback(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	state := r.FormValue("state")
	if state != oauthStateString {
		return errors.Errorf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return wrapErrorf(err, "oauthConf.Exchange() failed")
	}

	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	ghUser, _, err := client.Users.Get("")
	if err != nil {
		return wrapErrorf(err, "client.Users.Get() failed")
	}
	debug.Printf("Logged in as GitHub user: %s\n", *ghUser.Login)
	// Save user to DB.
	user, err := GetCreateUser(*ghUser.Login)
	if err != nil {
		return wrapErrorf(err, "error saving user to the database")
	}
	a.Session.Values[UIDSessionKey] = user.UID
	if err := a.Session.Save(r, w); err != nil {
		return wrapErrorf(err, "error saving session")
	}
	var q redirectQuery
	if err := SchemaDecoder.Decode(&q, r.URL.Query()); err != nil {
		return wrapError(err)
	}
	if q.Redirect != "" {
		u, err := url.QueryUnescape(q.Redirect)
		if err != nil {
			return wrapError(err)
		}
		return &HTTPRedirect{To: u, Code: http.StatusSeeOther}
	}
	return &HTTPRedirect{To: "/", Code: http.StatusSeeOther}
}

type saveEmailForm struct {
	Email string `schema:"email"`
}

func serveSaveEmail(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	if redirect := a.Authed(r); redirect != nil {
		return redirect
	}
	if err := r.ParseForm(); err != nil {
		return wrapErrorf(err, "error parsing form")
	}
	var form saveEmailForm
	err := SchemaDecoder.Decode(&form, r.PostForm)
	if err != nil {
		return wrapErrorf(err, "error decoding form")
	}
	if err := SetEmail(*a.User, form.Email); err != nil {
		return wrapErrorf(err, "error setting email")
	}
	return &HTTPRedirect{To: "/", Code: http.StatusSeeOther}
}

func serveUserStatsSVG(c web.C, w http.ResponseWriter, r *http.Request) error {
	gid, err := getParamInt(c, "group_id")
	if err != nil {
		return wrapError(err)
	}
	g, err := GetGroup(gid)
	if err != nil {
		return wrapError(err)
	}
	uid, err := getParamInt(c, "user_id")
	if err != nil {
		return wrapError(err)
	}
	u, err := GetUser(UserSpec{UID: uid})
	if err != nil {
		return wrapError(err)
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	if err := CreateStreakSVG(u, g, w); err != nil {
		return wrapError(err)
	}
	return nil
}

func serveGroupCreate(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	if redirect := a.Authed(r); redirect != nil {
		return redirect
	}
	g, err := CreateGroup(*a.User)
	if err != nil {
		return wrapError(err)
	}
	return &HTTPRedirect{To: GroupURL(g), Code: http.StatusSeeOther}
}

var groupTemplate = pongo2.Must(pongo2.FromFile("templates/group.html"))

type groupTemplateVars struct {
	Login           string
	Group           Group
	DayCommitGroups []DayCommitGroup
}

func serveGroup(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	if redirect := a.Authed(r); redirect != nil {
		return redirect
	}
	gid, err := getParamInt(c, "group_id")
	if err != nil {
		return wrapError(err)
	}
	g, err := GetGroup(gid)
	if err != nil {
		return wrapError(err)
	}
	cs, err := GetGroupAllCommits(g)
	if err != nil {
		return wrapError(err)
	}
	loc, err := GetGroupLocation(g)
	if err != nil {
		return wrapError(err)
	}
	// TODO(samertm): Check that the user is in the group.
	return RenderTemplate(groupTemplate, w, groupTemplateVars{
		Login:           a.User.Login,
		Group:           g,
		DayCommitGroups: DayCommitGroups(cs, loc),
	})
}

func serveGroupRefresh(c web.C, w http.ResponseWriter, r *http.Request) error {
	gid, err := getParamInt(c, "group_id")
	if err != nil {
		return err
	}
	// Now, refresh group commits..
	g, err := GetGroup(gid)
	if err != nil {
		return err
	}
	if err := UpdateGroupCommits(g); err != nil {
		return err
	}
	return nil
}

type groupJoinQuery struct {
	Key string `schema:"key"`
}

func serveGroupJoin(c web.C, w http.ResponseWriter, r *http.Request) error {
	a := NewApp(c)
	if redirect := a.Authed(r); redirect != nil {
		return redirect
	}
	gid, err := getParamInt(c, "group_id")
	if err != nil {
		return err
	}
	g, err := GetGroup(gid)
	if err != nil {
		return err
	}
	var q groupJoinQuery
	if err := SchemaDecoder.Decode(&q, r.URL.Query()); err != nil {
		return wrapError(err)
	}
	if q.Key != GroupSecretKey(g) {
		return errors.Errorf("key %q does not match secret key for group %d", q.Key, g.GID)
	}
	// Key matches, add user to g and redirect to the group page.
	if err := GroupAddUser(g, *a.User); err != nil {
		return wrapError(err)
	}
	return &HTTPRedirect{To: GroupURL(g), Code: http.StatusSeeOther}
}

var (
	oauthConf = oauth2.Config{
		ClientID:     conf.Config.GitHubID,
		ClientSecret: conf.Config.GitHubSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
	oauthStateString = conf.Config.OAuthStateString
)

// Pass nil for transport.
func UnauthedGitHubClient(transport http.RoundTripper) *github.Client {
	t := github.UnauthenticatedRateLimitedTransport{
		ClientID:     oauthConf.ClientID,
		ClientSecret: oauthConf.ClientSecret,
		Transport:    transport,
	}
	return github.NewClient(t.Client())
}

func main() {
	// Initalize database.
	ExecuteSchemas()
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
	// TODO(samertm): Make this POST /user/email.
	goji.Post("/save_email", handler(serveSaveEmail))

	goji.Post("/group/create", handler(serveGroupCreate))
	goji.Post("/group/:group_id/refresh", handler(serveGroupRefresh))
	goji.Get("/group/:group_id/join", handler(serveGroupJoin))
	goji.Get("/group/:group_id", handler(serveGroup))
	goji.Get("/group/:group_id/user/:user_id/stats.svg", handler(serveUserStatsSVG))

	goji.Serve()
}
