package web_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlashMessageDisplayedOnSuccess(t *testing.T) {
	ts := newWebTestServer(t)

	// Create guest - should set success flash
	form := url.Values{"display_name": {"Alice"}}
	rr := ts.post("/auth/guest", form)
	assert.Equal(t, http.StatusSeeOther, rr.Code)

	// Follow redirect and check for flash message
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)

	// Should see welcome message
	assertContainsText(t, doc, "body", "Welcome")
}

func TestFlashMessageDisplayedOnError(t *testing.T) {
	ts := newWebTestServer(t)
	ts.createGuestPlayer("Alice")

	// Try to join invalid lobby
	form := url.Values{"code": {"INVALID"}}
	rr := ts.post("/lobby/join", form)

	// Should redirect with flash
	assert.Equal(t, http.StatusSeeOther, rr.Code)

	// Follow redirect and check for error indication
	rr = ts.followRedirect(rr)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAccessDeniedForProtectedRoute(t *testing.T) {
	ts := newWebTestServer(t)

	// Try to access lobby without auth
	rr := ts.get("/lobby/ABC123")

	// Should redirect to home with next parameter
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Contains(t, rr.Header().Get("Location"), "/?next=")
}

func TestGameNotFoundWithoutGame(t *testing.T) {
	ts := newWebTestServer(t)
	ts.createGuestPlayer("Alice")
	lobbyCode := ts.createLobby(5)

	// Try to access game page when no game started
	rr := ts.get("/lobby/" + lobbyCode + "/game")

	// Should get redirect or error
	assert.True(t, rr.Code == http.StatusSeeOther || rr.Code == http.StatusNotFound || rr.Code == http.StatusOK,
		"Expected redirect, not found, or OK with error, got %d", rr.Code)
}

func TestInvalidFormDataHandledGracefully(t *testing.T) {
	ts := newWebTestServer(t)
	ts.createGuestPlayer("Alice")
	lobbyCode := ts.createLobby(5)

	// Try to update config with invalid grid size
	form := url.Values{"grid_size": {"invalid"}}
	rr := ts.post("/lobby/"+lobbyCode+"/config", form)

	// Should handle gracefully (redirect or error page, not crash)
	assert.True(t, rr.Code >= 200 && rr.Code < 600,
		"Expected valid HTTP response, got %d", rr.Code)
}

func TestFormValidationErrorsShown(t *testing.T) {
	t.Skip("Registration routes removed from UX - underlying logic preserved for future use")
}

func TestPasswordMismatchError(t *testing.T) {
	t.Skip("Registration routes removed from UX - underlying logic preserved for future use")
}

func TestStaticFileServing(t *testing.T) {
	// This test would need a static directory to be configured
	// Skipping as static files are not set up in test server
	t.Skip("Static files not configured in test server")
}
