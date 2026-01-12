---
spec_id: "spec-005"
spec_name: "Web Interface Testing"
status: "PROPOSED"
---
# spec-005 - Web Interface Testing

## Overview

This specification defines an automated testing strategy for the web interface (HTML/HTMX). The goal is to provide fast, reliable tests that verify the web layer is correct without being overly coupled to specific UI implementation details.

The strategy favors **HTTP-level testing** over browser automation:
- Tests make HTTP requests and parse HTML responses
- Assertions focus on content and structure, not pixel-perfect rendering
- Fast execution (no browser startup), runs in CI easily
- HTMX fragments are tested directly via their HTTP responses

This approach aligns with the design principles: fast iteration loop, pragmatic testing, and building confidence in the web interface layer.

## Relevant context

- Design docs: `docs/design/code-design-tech-stack.md` - testing strategy section mentions "simplest and most pragmatic approach to automated web testing"
- Existing pattern: `internal/api/api_test.go` - HTTP-level testing with `httptest`
- Web interface: `internal/web/` - router, handlers, templates, SSE
- SSE tests: `internal/web/sse/*_test.go` - existing unit tests for hub/broadcaster

### Why HTTP-level testing over browser automation

| Approach | Speed | Complexity | What it tests | CI-friendly |
|----------|-------|------------|---------------|-------------|
| HTTP-level | Fast (~ms) | Low | Server responses, HTML structure | Yes |
| Browser (Playwright) | Slow (~s) | High | Full client experience, JS execution | Harder |

Browser automation introduces:
- Slower tests (browser startup, page loads, timeouts)
- Flaky tests (timing issues, resource loading)
- Complex setup (browser drivers, headless configuration)
- Harder debugging

HTTP-level testing is sufficient because:
- HTMX is a thin layer - if the server returns correct HTML with correct attributes, HTMX will work
- We test the server's behavior, which is where our logic lives
- SSE events are tested by verifying the server sends correct messages

Browser tests could be added later for critical SSE flows if needed, but should be minimal.

## Test Architecture

### Test Server Setup

Similar to `api_test.go`, we create a test server with the full web router:

```go
type webTestServer struct {
    handler http.Handler
    app     *factory.App
    cookies *cookieJar // For session persistence across requests
}

func newWebTestServer(t *testing.T) *webTestServer {
    t.Helper()

    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    app := factory.New(factory.Config{})

    // Load dictionary for game tests
    err := app.DictionaryService.LoadFromFile(t.Context(), "../../data/words.txt")
    require.NoError(t, err)

    router := web.NewRouter(web.RouterConfig{
        Logger:          logger,
        AuthService:     app.AuthService,
        LobbyController: app.LobbyController,
        GameController:  app.GameController,
        BoardService:    app.BoardService,
        ScoringService:  app.ScoringService,
        HubManager:      app.HubManager,
        StaticDir:       "../../internal/web/static",
    })

    return &webTestServer{
        handler: router,
        app:     app,
        cookies: newCookieJar(),
    }
}
```

### Cookie Jar for Sessions

Web tests need to maintain session cookies across requests:

```go
type cookieJar struct {
    cookies map[string]*http.Cookie
}

func (ts *webTestServer) request(method, path string, form url.Values) *httptest.ResponseRecorder {
    var body io.Reader
    if form != nil {
        body = strings.NewReader(form.Encode())
    }

    req := httptest.NewRequest(method, path, body)
    if form != nil {
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    }

    // Add session cookie from jar
    ts.cookies.addTo(req)

    rr := httptest.NewRecorder()
    ts.handler.ServeHTTP(rr, req)

    // Extract Set-Cookie headers into jar
    ts.cookies.extract(rr)

    return rr
}
```

### HTML Parsing with goquery

Use `goquery` for jQuery-like HTML querying:

```go
import "github.com/PuerkitoBio/goquery"

func parseHTML(r io.Reader) *goquery.Document {
    doc, _ := goquery.NewDocumentFromReader(r)
    return doc
}

// Example assertions:
doc := parseHTML(rr.Body)
assert.True(t, doc.Find("#member-list").Length() > 0)
assert.Contains(t, doc.Find("#member-list").Text(), "Alice")
assert.True(t, doc.Find("button[hx-post='/lobby/ABC/game/start']").Length() > 0)
```

## Test Categories

### 1. Auth Flow Tests

Test file: `internal/web/web_auth_test.go`

| Test | Description |
|------|-------------|
| `TestGuestCreation` | POST /auth/guest creates session, redirects to home |
| `TestLogin` | POST /auth/login with valid credentials sets session |
| `TestLoginInvalidCredentials` | POST /auth/login shows error on invalid credentials |
| `TestRegister` | POST /auth/register creates account, sets session |
| `TestRegisterDuplicateUsername` | Shows error for existing username |
| `TestLogout` | POST /auth/logout clears session cookie |
| `TestProtectedRouteRedirect` | /lobby/ABC without session redirects to login |
| `TestSessionPersistence` | Session cookie works across requests |

### 2. Lobby Flow Tests

Test file: `internal/web/web_lobby_test.go`

| Test | Description |
|------|-------------|
| `TestCreateLobby` | POST /lobby creates lobby, redirects to /lobby/{code} |
| `TestJoinLobby` | POST /lobby/join with code joins and redirects |
| `TestJoinLobbyInvalidCode` | Shows error for invalid code |
| `TestLobbyPageShowsMembers` | GET /lobby/{code} shows member list |
| `TestLobbyPageShowsConfig` | Shows grid size configuration |
| `TestHostSeesControls` | Host sees start game, config controls |
| `TestNonHostNoControls` | Non-host doesn't see host-only controls |
| `TestLeaveLobby` | POST /lobby/{code}/leave removes player, redirects |
| `TestUpdateConfig` | PATCH config updates grid size (host only) |
| `TestUpdateConfigNonHost` | Non-host cannot update config |

### 3. Game Flow Tests

Test file: `internal/web/web_game_test.go`

| Test | Description |
|------|-------------|
| `TestStartGame` | POST /lobby/{code}/game/start redirects to game page |
| `TestGamePageShowsBoard` | GET /lobby/{code}/game shows player's board |
| `TestAnnouncerSeesLetterPicker` | Announcer sees letter input |
| `TestNonAnnouncerNoLetterPicker` | Non-announcer doesn't see letter picker |
| `TestAnnounceLetter` | POST announce updates game state |
| `TestPlaceLetter` | POST place updates board |
| `TestPlaceOnOccupiedCell` | Error when placing on filled cell |
| `TestGameComplete` | Shows scores when game finishes |
| `TestAbandonGame` | Host can abandon, redirects to lobby |

### 4. HTMX Fragment Tests

Test file: `internal/web/web_htmx_test.go`

Verify HTMX endpoints return correct fragments:

| Test | Description |
|------|-------------|
| `TestMemberListFragment` | OOB swap fragment has correct structure |
| `TestPlacementStatusFragment` | Shows "X/Y players placed" |
| `TestLobbyControlsFragment` | Contains correct form actions |
| `TestFragmentsHaveOOBAttribute` | Verify `hx-swap-oob="true"` present |

### 5. Error Handling Tests

Test file: `internal/web/web_error_test.go`

| Test | Description |
|------|-------------|
| `TestNotFoundLobby` | GET /lobby/INVALID shows error |
| `TestFlashMessages` | Flash message displayed and cleared |
| `TestFormValidationErrors` | Invalid form shows inline errors |

## Assertion Patterns

### Testing for Element Presence

Use IDs and semantic selectors that are stable:

```go
// Good - uses ID
assert.True(t, doc.Find("#member-list").Length() > 0)

// Good - uses data attribute
assert.True(t, doc.Find("[data-testid='start-game-btn']").Length() > 0)

// Avoid - brittle selector
assert.True(t, doc.Find("div.card > div.card-body > button.btn-primary").Length() > 0)
```

### Testing Content

Assert on text content, not exact HTML:

```go
// Good
assert.Contains(t, doc.Find("#member-list").Text(), "Alice")

// Good - verify role badge
memberItem := doc.Find("#member-list .member-item").First()
assert.Contains(t, memberItem.Text(), "Host")

// Avoid - exact HTML matching
assert.Equal(t, "<div class=\"member\">Alice (Host)</div>", html)
```

### Testing HTMX Attributes

Verify HTMX integration without testing HTMX itself:

```go
// Verify form has correct HTMX attributes
form := doc.Find("#create-lobby-form")
assert.Equal(t, "/lobby", form.AttrOr("hx-post", ""))
assert.Equal(t, "body", form.AttrOr("hx-target", ""))

// Verify OOB fragment
assert.Equal(t, "true", doc.Find("#member-list").AttrOr("hx-swap-oob", ""))
```

### Testing Redirects

```go
rr := ts.request("POST", "/lobby", form)
assert.Equal(t, http.StatusSeeOther, rr.Code)
assert.Contains(t, rr.Header().Get("Location"), "/lobby/")
```

## Test Data Attributes

To make tests more stable, templates should include `data-testid` attributes on key elements:

```html
<button data-testid="start-game-btn" hx-post="/lobby/{{.Code}}/game/start">
  Start Game
</button>

<div id="member-list" data-testid="member-list">
  <!-- members -->
</div>
```

This allows tests to use `[data-testid='...']` selectors which are:
- Clearly for testing purposes
- Stable across UI refactoring
- Won't conflict with styling changes

## Package Structure

```
internal/web/
├── web_test.go          # Test helpers (webTestServer, cookieJar, parseHTML)
├── web_auth_test.go     # Auth flow tests
├── web_lobby_test.go    # Lobby flow tests
├── web_game_test.go     # Game flow tests
├── web_htmx_test.go     # HTMX fragment tests
├── web_error_test.go    # Error handling tests
└── testdata/            # Any test fixtures if needed
```

## Task implementation strategy

1. **Add goquery dependency and test infrastructure**
   - Add `github.com/PuerkitoBio/goquery` dependency
   - Create `web_test.go` with `webTestServer`, cookie jar, helper functions
   - Verify test server can be created and responds

2. **Implement auth flow tests**
   - Guest creation, login, register, logout tests
   - Session persistence test
   - Protected route redirect test

3. **Add data-testid attributes to templates**
   - Add `data-testid` to key elements used in tests
   - Start game button, member list, lobby controls, game board, etc.

4. **Implement lobby flow tests**
   - Create/join lobby tests
   - Member list visibility
   - Host controls visibility
   - Config update tests

5. **Implement game flow tests**
   - Start game, announce, place tests
   - Announcer vs non-announcer visibility
   - Game completion scores display

6. **Implement HTMX fragment tests**
   - Verify OOB swap attributes
   - Test fragment content

7. **Implement error handling tests**
   - 404 handling, flash messages, form validation

8. **CI integration**
   - Ensure tests run in CI with `task test`
   - Add any necessary test timeouts or configurations

## Future Considerations (Out of Scope)

### Browser Tests

If HTTP-level tests prove insufficient for SSE flows, we could add minimal browser tests using Playwright or Rod:

```go
// Example with rod (Go library for browser automation)
func TestSSEMemberUpdate(t *testing.T) {
    // Start test server
    // Launch headless browser
    // Navigate to lobby page
    // In separate goroutine, have another player join
    // Assert member list updates without refresh
}
```

This should only be done if:
1. HTTP-level tests are complete and passing
2. There are specific SSE bugs that can't be caught otherwise
3. We're willing to accept slower test times

### Visual Regression Testing

Not planned, but could use tools like Percy or Playwright's screenshot comparison if UI consistency becomes important.

## Status details

Status: PROPOSED - Awaiting approval before implementation.
