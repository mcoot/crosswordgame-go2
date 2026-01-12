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
	ts := newWebTestServer(t)

	// Register new user
	form := url.Values{
		"username":         {"alice"},
		"password":         {"secret123"},
		"password_confirm": {"secret123"},
		"display_name":     {"Alice"},
	}
	rr := ts.post("/auth/register", form)

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	// Session cookie should be set
	assert.True(t, ts.cookies.hasSession())

	// Follow redirect and verify logged in
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "nav", "Alice")
}

func TestRegisterDuplicateUsername(t *testing.T) {
	ts := newWebTestServer(t)

	// Register first user
	form := url.Values{
		"username":         {"alice"},
		"password":         {"secret123"},
		"password_confirm": {"secret123"},
		"display_name":     {"Alice"},
	}
	rr := ts.post("/auth/register", form)
	assert.Equal(t, http.StatusSeeOther, rr.Code)

	// Clear session to register second user
	ts.cookies = newCookieJar()

	// Try to register with same username
	form = url.Values{
		"username":         {"alice"},
		"password":         {"different456"},
		"password_confirm": {"different456"},
		"display_name":     {"Alice2"},
	}
	rr = ts.post("/auth/register", form)

	// Should re-render page with error (200 status, not redirect)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Page should show error message about username
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "body", "already taken")

	// Session should NOT be set
	assert.False(t, ts.cookies.hasSession())
}

func TestLogin(t *testing.T) {
	ts := newWebTestServer(t)

	// First register a user
	registerForm := url.Values{
		"username":         {"bob"},
		"password":         {"secret123"},
		"password_confirm": {"secret123"},
		"display_name":     {"Bob"},
	}
	ts.post("/auth/register", registerForm)

	// Clear session
	ts.cookies = newCookieJar()
	assert.False(t, ts.cookies.hasSession())

	// Login
	loginForm := url.Values{
		"username": {"bob"},
		"password": {"secret123"},
	}
	rr := ts.post("/auth/login", loginForm)

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	// Session should be set
	assert.True(t, ts.cookies.hasSession())

	// Verify logged in
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "nav", "Bob")
}

func TestLoginInvalidCredentials(t *testing.T) {
	ts := newWebTestServer(t)

	// First register a user
	registerForm := url.Values{
		"username":         {"charlie"},
		"password":         {"secret123"},
		"password_confirm": {"secret123"},
		"display_name":     {"Charlie"},
	}
	ts.post("/auth/register", registerForm)

	// Clear session
	ts.cookies = newCookieJar()

	// Try login with wrong password
	loginForm := url.Values{
		"username": {"charlie"},
		"password": {"wrongpassword"},
	}
	rr := ts.post("/auth/login", loginForm)

	// Should re-render login page with error (200 status, not redirect)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Page should show error message
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "body", "Invalid username or password")

	// Session should NOT be set
	assert.False(t, ts.cookies.hasSession())
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

	// Verify logged out - should see login link
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	assertContainsElement(t, doc, "a[href='/login']")
}

func TestProtectedRouteRedirect(t *testing.T) {
	ts := newWebTestServer(t)

	// Try to access lobby page without auth
	rr := ts.get("/lobby/ABC123")

	// Should redirect to login with next parameter
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	location := rr.Header().Get("Location")
	assert.Contains(t, location, "/login")
	assert.Contains(t, location, "next=")
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
	ts := newWebTestServer(t)

	rr := ts.get("/login")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should have login form
	assertContainsElement(t, doc, "form[action='/auth/login']")
	assertContainsElement(t, doc, "input[name='username']")
	assertContainsElement(t, doc, "input[name='password']")
}

func TestRegisterPage(t *testing.T) {
	ts := newWebTestServer(t)

	rr := ts.get("/register")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should have register form
	assertContainsElement(t, doc, "form[action='/auth/register']")
	assertContainsElement(t, doc, "input[name='username']")
	assertContainsElement(t, doc, "input[name='password']")
	assertContainsElement(t, doc, "input[name='display_name']")
}

func TestHomePage(t *testing.T) {
	ts := newWebTestServer(t)

	rr := ts.get("/")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should have guest creation form
	assertContainsElement(t, doc, "form[action='/auth/guest']")
	// Should have login/register links
	assertContainsElement(t, doc, "a[href='/login']")
	assertContainsElement(t, doc, "a[href='/register']")
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
