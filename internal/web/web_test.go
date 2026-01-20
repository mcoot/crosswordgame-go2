package web_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"

	"github.com/mcoot/crosswordgame-go2/internal/factory"
	"github.com/mcoot/crosswordgame-go2/internal/web"
)

// webTestServer provides a test server for web interface testing
type webTestServer struct {
	t       *testing.T
	handler http.Handler
	app     *factory.App
	cookies *cookieJar
}

// newWebTestServer creates a new test server with all dependencies wired
func newWebTestServer(t *testing.T) *webTestServer {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, err := factory.New(factory.Config{})
	require.NoError(t, err)

	// Load dictionary for game tests
	err = app.DictionaryService.LoadFromFile(t.Context(), "../../data/words.txt")
	require.NoError(t, err)

	router := web.NewRouter(web.RouterConfig{
		Logger:          logger,
		AuthService:     app.AuthService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		ScoringService:  app.ScoringService,
		HubManager:      app.HubManager,
		StaticDir:       "", // No static files in tests
	})

	return &webTestServer{
		t:       t,
		handler: router,
		app:     app,
		cookies: newCookieJar(),
	}
}

// request makes an HTTP request and returns the response
func (ts *webTestServer) request(method, path string, form url.Values, htmx bool) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if htmx {
		req.Header.Set("HX-Request", "true")
	}

	// Add cookies from jar
	ts.cookies.addTo(req)

	rr := httptest.NewRecorder()
	ts.handler.ServeHTTP(rr, req)

	// Extract Set-Cookie headers into jar
	ts.cookies.extract(rr)

	return rr
}

// get makes a GET request
func (ts *webTestServer) get(path string) *httptest.ResponseRecorder {
	return ts.request(http.MethodGet, path, nil, false)
}

// post makes a POST request with form data (non-HTMX)
func (ts *webTestServer) post(path string, form url.Values) *httptest.ResponseRecorder {
	return ts.request(http.MethodPost, path, form, false)
}

// postHTMX makes a POST request with form data as an HTMX request
func (ts *webTestServer) postHTMX(path string, form url.Values) *httptest.ResponseRecorder {
	return ts.request(http.MethodPost, path, form, true)
}

// parseHTML parses the response body as HTML
func parseHTML(r io.Reader) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		panic(err)
	}
	return doc
}

// cookieJar maintains cookies across requests (like a browser would)
type cookieJar struct {
	cookies map[string]*http.Cookie
}

func newCookieJar() *cookieJar {
	return &cookieJar{
		cookies: make(map[string]*http.Cookie),
	}
}

// addTo adds all cookies to the request
func (j *cookieJar) addTo(req *http.Request) {
	for _, cookie := range j.cookies {
		req.AddCookie(cookie)
	}
}

// extract extracts Set-Cookie headers from response
func (j *cookieJar) extract(rr *httptest.ResponseRecorder) {
	for _, cookie := range rr.Result().Cookies() {
		if cookie.MaxAge < 0 {
			// Cookie being deleted
			delete(j.cookies, cookie.Name)
		} else {
			j.cookies[cookie.Name] = cookie
		}
	}
}

// hasSession returns true if the session cookie is set
func (j *cookieJar) hasSession() bool {
	_, ok := j.cookies["session"]
	return ok
}

// Helper functions for common test operations

// createGuestPlayer creates a guest player and returns the display name
func (ts *webTestServer) createGuestPlayer(displayName string) {
	ts.t.Helper()
	form := url.Values{"display_name": {displayName}}
	rr := ts.post("/auth/guest", form)
	require.Equal(ts.t, http.StatusSeeOther, rr.Code, "Expected redirect after guest creation")
	require.True(ts.t, ts.cookies.hasSession(), "Expected session cookie to be set")
}

// createRegisteredPlayer creates a registered player directly via the auth service
// and sets up the session cookie for subsequent requests
// Preserved for future re-enablement of login functionality
//
//nolint:unused
func (ts *webTestServer) createRegisteredPlayer(username, password, displayName string) {
	ts.t.Helper()
	session, err := ts.app.AuthService.RegisterPlayer(ts.t.Context(), username, password, displayName)
	require.NoError(ts.t, err, "Expected registration to succeed")
	// Set the session cookie
	ts.cookies.cookies["session"] = &http.Cookie{
		Name:  "session",
		Value: session.Token,
	}
}

// createLobby creates a lobby and returns the lobby code
func (ts *webTestServer) createLobby(gridSize int) string {
	ts.t.Helper()
	form := url.Values{"grid_size": {strconv.Itoa(gridSize)}}
	rr := ts.post("/lobby", form)
	require.Equal(ts.t, http.StatusSeeOther, rr.Code, "Expected redirect after lobby creation")

	// Extract lobby code from redirect location
	location := rr.Header().Get("Location")
	require.Contains(ts.t, location, "/lobby/", "Expected redirect to lobby page")

	// Extract code from /lobby/{code}
	parts := strings.Split(location, "/lobby/")
	require.Len(ts.t, parts, 2, "Expected location to contain /lobby/{code}")
	return parts[1]
}

// joinLobby joins a lobby by code
func (ts *webTestServer) joinLobby(code string) {
	ts.t.Helper()
	form := url.Values{"code": {code}}
	rr := ts.post("/lobby/join", form)
	require.Equal(ts.t, http.StatusSeeOther, rr.Code, "Expected redirect after joining lobby")
}

// startGame starts a game in the given lobby (uses HTMX request)
func (ts *webTestServer) startGame(lobbyCode string) {
	ts.t.Helper()
	rr := ts.postHTMX("/lobby/"+lobbyCode+"/game/start", nil)
	require.Equal(ts.t, http.StatusNoContent, rr.Code, "Expected 204 No Content after starting game")
	require.NotEmpty(ts.t, rr.Header().Get("HX-Redirect"), "Expected HX-Redirect header after starting game")
}

// followRedirect follows a redirect and returns the response
// Works with both traditional Location headers and HTMX HX-Redirect headers
func (ts *webTestServer) followRedirect(rr *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	ts.t.Helper()
	// Check for HTMX redirect first
	location := rr.Header().Get("HX-Redirect")
	if location == "" {
		// Fall back to traditional redirect
		location = rr.Header().Get("Location")
	}
	require.NotEmpty(ts.t, location, "Expected Location or HX-Redirect header for redirect")
	return ts.get(location)
}

// Assertion helpers

// assertContainsElement asserts that the document contains an element matching the selector
func assertContainsElement(t *testing.T, doc *goquery.Document, selector string) {
	t.Helper()
	if doc.Find(selector).Length() == 0 {
		t.Errorf("Expected to find element matching %q, but none found", selector)
	}
}

// assertNotContainsElement asserts that the document does not contain an element matching the selector
func assertNotContainsElement(t *testing.T, doc *goquery.Document, selector string) {
	t.Helper()
	if doc.Find(selector).Length() > 0 {
		t.Errorf("Expected NOT to find element matching %q, but found %d", selector, doc.Find(selector).Length())
	}
}

// assertContainsText asserts that the element matching the selector contains the text
func assertContainsText(t *testing.T, doc *goquery.Document, selector, text string) {
	t.Helper()
	el := doc.Find(selector)
	if el.Length() == 0 {
		t.Errorf("Expected to find element matching %q, but none found", selector)
		return
	}
	if !strings.Contains(el.Text(), text) {
		t.Errorf("Expected element %q to contain %q, but got %q", selector, text, el.Text())
	}
}
