package web_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuestCreation(t *testing.T) {
	ts := newWebTestServer(t)

	// Create guest player
	form := url.Values{"display_name": {"Alice"}}
	rr := ts.post("/auth/guest", form)

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	// Session cookie should be set
	assert.True(t, ts.cookies.hasSession())

	// Follow redirect and check we're logged in
	rr = ts.followRedirect(rr)
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should show player name in nav
	assertContainsText(t, doc, "nav", "Alice")
	// Should show lobby creation form (authenticated user)
	assertContainsElement(t, doc, "form[action='/lobby']")
}

func TestGuestCreationEmptyName(t *testing.T) {
	ts := newWebTestServer(t)

	// Try to create guest with empty name
	form := url.Values{"display_name": {""}}
	rr := ts.post("/auth/guest", form)

	// Should redirect back (error via flash message)
	assert.Equal(t, http.StatusSeeOther, rr.Code)

	// Session should NOT be set
	assert.False(t, ts.cookies.hasSession())
}

func TestRegister(t *testing.T) {
	t.Skip("Registration routes removed from UX - underlying logic preserved for future use")
}

func TestRegisterDuplicateUsername(t *testing.T) {
	t.Skip("Registration routes removed from UX - underlying logic preserved for future use")
}

func TestLogin(t *testing.T) {
	t.Skip("Login routes removed from UX - underlying logic preserved for future use")
}

func TestLoginInvalidCredentials(t *testing.T) {
	t.Skip("Login routes removed from UX - underlying logic preserved for future use")
}

func TestLogout(t *testing.T) {
	ts := newWebTestServer(t)

	// Create guest
	ts.createGuestPlayer("Dave")
	assert.True(t, ts.cookies.hasSession())

	// Logout
	rr := ts.post("/auth/logout", nil)

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	// Session should be cleared
	assert.False(t, ts.cookies.hasSession())

	// Verify logged out - should see guest form
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	assertContainsElement(t, doc, "form[action='/auth/guest']")
}

func TestProtectedRouteRedirect(t *testing.T) {
	ts := newWebTestServer(t)

	// Try to access lobby page without auth
	rr := ts.get("/lobby/ABC123")

	// Should redirect to home with next parameter
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	location := rr.Header().Get("Location")
	assert.Contains(t, location, "/?next=")
}

func TestSessionPersistence(t *testing.T) {
	ts := newWebTestServer(t)

	// Create guest
	ts.createGuestPlayer("Eve")

	// Make multiple requests - session should persist
	rr1 := ts.get("/")
	doc1 := parseHTML(rr1.Body)
	assertContainsText(t, doc1, "nav", "Eve")

	rr2 := ts.get("/")
	doc2 := parseHTML(rr2.Body)
	assertContainsText(t, doc2, "nav", "Eve")

	// Both requests should see the same user
	assert.Equal(t, http.StatusOK, rr1.Code)
	assert.Equal(t, http.StatusOK, rr2.Code)
}

func TestLoginPage(t *testing.T) {
	t.Skip("Login routes removed from UX - underlying logic preserved for future use")
}

func TestRegisterPage(t *testing.T) {
	t.Skip("Registration routes removed from UX - underlying logic preserved for future use")
}

func TestHomePage(t *testing.T) {
	ts := newWebTestServer(t)

	rr := ts.get("/")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should have guest creation form
	assertContainsElement(t, doc, "form[action='/auth/guest']")
	// Login and register links removed from UX
	assertNotContainsElement(t, doc, "a[href='/login']")
	assertNotContainsElement(t, doc, "a[href='/register']")
}

func TestHomePageAuthenticated(t *testing.T) {
	ts := newWebTestServer(t)

	// Create guest
	ts.createGuestPlayer("Frank")

	rr := ts.get("/")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should have lobby creation form
	assertContainsElement(t, doc, "form[action='/lobby']")
	// Should have join lobby form
	assertContainsElement(t, doc, "form[action='/lobby/join']")
}
