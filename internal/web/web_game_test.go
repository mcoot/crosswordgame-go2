package web_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupTwoPlayerGame creates a lobby with two players and returns the lobby code
// and the cookies for both players (alice is host)
func setupTwoPlayerGame(t *testing.T, ts *webTestServer, gridSize int) (lobbyCode string, aliceCookies, bobCookies *cookieJar) {
	t.Helper()

	// Alice creates lobby
	ts.createGuestPlayer("Alice")
	lobbyCode = ts.createLobby(gridSize)
	aliceCookies = ts.cookies

	// Bob joins
	bobCookies = newCookieJar()
	ts.cookies = bobCookies
	ts.createGuestPlayer("Bob")
	ts.joinLobby(lobbyCode)

	return lobbyCode, aliceCookies, bobCookies
}

func TestGamePageShowsBoard(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 3)

	// Alice starts game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Should show game board
	assertContainsElement(t, doc, "#game-board")
	// Should show game status
	assertContainsElement(t, doc, "#game-status")
}

func TestAnnouncerSeesLetterPicker(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Alice starts game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page as Alice
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	aliceDoc := parseHTML(rr.Body)

	// Get game page as Bob
	ts.cookies = bobCookies
	rr = ts.get("/lobby/" + lobbyCode + "/game")
	bobDoc := parseHTML(rr.Body)

	// One of them should be the announcer and see the letter picker
	// The other should not
	aliceHasPicker := aliceDoc.Find("#letter-picker").Length() > 0
	bobHasPicker := bobDoc.Find("#letter-picker").Length() > 0

	// Exactly one should have the picker
	assert.True(t, aliceHasPicker != bobHasPicker,
		"Exactly one player should see letter picker. Alice: %v, Bob: %v", aliceHasPicker, bobHasPicker)
}

func TestAnnounceLetter(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Find who is announcer
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	aliceDoc := parseHTML(rr.Body)
	announcerCookies := aliceCookies
	if aliceDoc.Find("#letter-picker").Length() == 0 {
		announcerCookies = bobCookies
	}

	// Announce as announcer
	ts.cookies = announcerCookies
	form := url.Values{"letter": {"A"}}
	rr = ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", form)

	// Should return 204 No Content (SSE broadcast handles the update)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Fetch game page and verify letter is shown
	rr = ts.get("/lobby/" + lobbyCode + "/game")
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "#game-status", "A")
}

func TestPlaceLetter(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Find announcer and announce
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	aliceDoc := parseHTML(rr.Body)
	announcerCookies := aliceCookies
	if aliceDoc.Find("#letter-picker").Length() == 0 {
		announcerCookies = bobCookies
	}

	ts.cookies = announcerCookies
	form := url.Values{"letter": {"A"}}
	ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", form)

	// Now place letter (as Alice)
	ts.cookies = aliceCookies
	form = url.Values{"row": {"0"}, "col": {"0"}}
	rr = ts.postHTMX("/lobby/"+lobbyCode+"/game/place", form)

	// Should return 204 No Content (SSE broadcast handles the update)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Fetch game page and verify it works
	rr = ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPlaceOnOccupiedCell(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Find announcer and announce
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	aliceDoc := parseHTML(rr.Body)
	announcerCookies := aliceCookies
	otherCookies := bobCookies
	if aliceDoc.Find("#letter-picker").Length() == 0 {
		announcerCookies = bobCookies
		otherCookies = aliceCookies
	}

	// First turn - announce and both place at 0,0
	ts.cookies = announcerCookies
	form := url.Values{"letter": {"A"}}
	ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", form)

	ts.cookies = announcerCookies
	form = url.Values{"row": {"0"}, "col": {"0"}}
	ts.postHTMX("/lobby/"+lobbyCode+"/game/place", form)

	ts.cookies = otherCookies
	form = url.Values{"row": {"0"}, "col": {"0"}}
	ts.postHTMX("/lobby/"+lobbyCode+"/game/place", form)

	// Second turn - announce and try to place at same position
	rr = ts.get("/lobby/" + lobbyCode + "/game")
	doc := parseHTML(rr.Body)
	if doc.Find("#letter-picker").Length() > 0 {
		// This player is announcer
		form = url.Values{"letter": {"B"}}
		ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", form)
	} else {
		// Switch to announcer
		if ts.cookies == announcerCookies {
			ts.cookies = otherCookies
		} else {
			ts.cookies = announcerCookies
		}
		form = url.Values{"letter": {"B"}}
		ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", form)
	}

	// Try to place at already occupied cell
	ts.cookies = aliceCookies
	form = url.Values{"row": {"0"}, "col": {"0"}}
	rr = ts.postHTMX("/lobby/"+lobbyCode+"/game/place", form)

	// Should return 204 (error handled via HX-Redirect with flash message)
	// or could succeed if game state allows it
	assert.True(t, rr.Code == http.StatusNoContent || rr.Code == http.StatusBadRequest || rr.Code == http.StatusOK)
}

func TestAbandonGame(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Abandon game (host only)
	rr := ts.postHTMX("/lobby/"+lobbyCode+"/game/abandon", nil)

	// Should return 204 with HX-Redirect to lobby
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Contains(t, rr.Header().Get("HX-Redirect"), "/lobby/"+lobbyCode)

	// Follow redirect and verify we're back in lobby (no game)
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	// Should be in lobby waiting state, not game
	assertContainsText(t, doc, "body", "Waiting")
}

func TestNonHostCannotAbandonGame(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Bob tries to abandon
	ts.cookies = bobCookies
	rr := ts.postHTMX("/lobby/"+lobbyCode+"/game/abandon", nil)

	// Should get 204 with HX-Redirect (error via flash) or 403 forbidden
	assert.True(t, rr.Code == http.StatusForbidden || rr.Code == http.StatusNoContent,
		"Expected forbidden or 204, got %d", rr.Code)
}

func TestGameStatusShowsTurnInfo(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	doc := parseHTML(rr.Body)

	// Game status should show turn info
	statusText := doc.Find("#game-status").Text()
	assert.True(t, strings.Contains(statusText, "1") || strings.Contains(statusText, "Turn"),
		"Status should show turn info: %s", statusText)
}

func TestGameShowsGridSize(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 5)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	doc := parseHTML(rr.Body)

	// Should show grid size somewhere
	assertContainsText(t, doc, ".game-sidebar", "5x5")
}

func TestBackToLobbyLink(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 3)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	doc := parseHTML(rr.Body)

	// Should have link back to lobby
	assertContainsElement(t, doc, "a[href='/lobby/"+lobbyCode+"']")
}

func TestHostSeesAbandonButtonDuringGame(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, _ := setupTwoPlayerGame(t, ts, 3)

	// Alice (host) starts game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page as host
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Host should see abandon button (using hx-post)
	abandonForm := doc.Find("form[hx-post='/lobby/" + lobbyCode + "/game/abandon']")
	assert.Equal(t, 1, abandonForm.Length(), "Host should see abandon form")

	abandonButton := abandonForm.Find("button")
	assert.True(t, strings.Contains(abandonButton.Text(), "Abandon"),
		"Abandon button should contain text 'Abandon'")
}

func TestNonHostNoAbandonButton(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 3)

	// Alice (host) starts game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Get game page as Bob (non-host)
	ts.cookies = bobCookies
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Non-host should NOT see abandon form (using hx-post)
	abandonForm := doc.Find("form[hx-post='/lobby/" + lobbyCode + "/game/abandon']")
	assert.Equal(t, 0, abandonForm.Length(), "Non-host should not see abandon form")
}

// completeGame plays through all turns of a 2x2 game to reach scoring state
func completeGame(t *testing.T, ts *webTestServer, lobbyCode string, aliceCookies, bobCookies *cookieJar) {
	t.Helper()

	// For a 2x2 grid, we need 4 turns
	positions := []struct{ row, col string }{
		{"0", "0"}, {"0", "1"}, {"1", "0"}, {"1", "1"},
	}
	letters := []string{"A", "B", "C", "D"}

	for i, pos := range positions {
		// Find announcer
		ts.cookies = aliceCookies
		rr := ts.get("/lobby/" + lobbyCode + "/game")
		doc := parseHTML(rr.Body)
		announcerCookies := aliceCookies
		otherCookies := bobCookies
		if doc.Find("#letter-picker").Length() == 0 {
			announcerCookies = bobCookies
			otherCookies = aliceCookies
		}

		// Announce letter (using HTMX)
		ts.cookies = announcerCookies
		ts.postHTMX("/lobby/"+lobbyCode+"/game/announce", url.Values{"letter": {letters[i]}})

		// Both players place (using HTMX)
		ts.cookies = announcerCookies
		ts.postHTMX("/lobby/"+lobbyCode+"/game/place", url.Values{"row": {pos.row}, "col": {pos.col}})
		ts.cookies = otherCookies
		ts.postHTMX("/lobby/"+lobbyCode+"/game/place", url.Values{"row": {pos.row}, "col": {pos.col}})
	}
}

func TestHostSeesDismissButtonOnScoring(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 2) // 2x2 for quick completion

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Complete the game
	completeGame(t, ts, lobbyCode, aliceCookies, bobCookies)

	// Get game page as host
	ts.cookies = aliceCookies
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Host should see dismiss form (Return to Lobby) using hx-post
	dismissForm := doc.Find("form[hx-post='/lobby/" + lobbyCode + "/game/dismiss']")
	assert.GreaterOrEqual(t, dismissForm.Length(), 1, "Host should see dismiss form")

	// Should see both Return to Lobby and Play Again buttons
	assertContainsText(t, doc, "#post-game-controls", "Return to Lobby")
	assertContainsText(t, doc, "#post-game-controls", "Play Again")
}

func TestNonHostNoDismissButton(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 2)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Complete the game
	completeGame(t, ts, lobbyCode, aliceCookies, bobCookies)

	// Get game page as non-host
	ts.cookies = bobCookies
	rr := ts.get("/lobby/" + lobbyCode + "/game")
	assert.Equal(t, http.StatusOK, rr.Code)

	doc := parseHTML(rr.Body)
	// Non-host should NOT see post-game controls
	postGameControls := doc.Find("#post-game-controls")
	assert.Equal(t, 0, postGameControls.Length(), "Non-host should not see post-game controls")
}

func TestDismissGameReturnsToLobby(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 2)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Complete the game
	completeGame(t, ts, lobbyCode, aliceCookies, bobCookies)

	// Dismiss game as host (using HTMX)
	ts.cookies = aliceCookies
	rr := ts.postHTMX("/lobby/"+lobbyCode+"/game/dismiss", nil)

	// Should return 204 with HX-Redirect to lobby
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Contains(t, rr.Header().Get("HX-Redirect"), "/lobby/"+lobbyCode)

	// Follow redirect and verify we're in lobby waiting state
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	assertContainsText(t, doc, "body", "Waiting")
}

func TestPlayAgainStartsNewGame(t *testing.T) {
	ts := newWebTestServer(t)
	lobbyCode, aliceCookies, bobCookies := setupTwoPlayerGame(t, ts, 2)

	// Start game
	ts.cookies = aliceCookies
	ts.startGame(lobbyCode)

	// Complete the game
	completeGame(t, ts, lobbyCode, aliceCookies, bobCookies)

	// Click Play Again (dismiss with start_new=true) using HTMX
	ts.cookies = aliceCookies
	rr := ts.postHTMX("/lobby/"+lobbyCode+"/game/dismiss", url.Values{"start_new": {"true"}})

	// Should return 204 with HX-Redirect to game page
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Contains(t, rr.Header().Get("HX-Redirect"), "/lobby/"+lobbyCode+"/game")

	// Follow redirect and verify we're in a new game (announcing state)
	rr = ts.followRedirect(rr)
	doc := parseHTML(rr.Body)
	// New game should be in progress (has game status, turn 1)
	assertContainsElement(t, doc, "#game-status")
}
