package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcoot/crosswordgame-go2/internal/api"
	"github.com/mcoot/crosswordgame-go2/internal/api/response"
	"github.com/mcoot/crosswordgame-go2/internal/factory"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
)

// testServer creates a test server with all dependencies
type testServer struct {
	handler http.Handler
	storage *memory.Storage
	auth    *auth.Service
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// API tests are integration tests - use production factory with real random/clock
	app, err := factory.New(factory.Config{})
	require.NoError(t, err)
	err = app.DictionaryService.LoadFromFile(t.Context(), "../../data/words.txt")
	require.NoError(t, err)

	router := api.NewRouter(api.RouterConfig{
		Logger:          logger,
		AuthService:     app.AuthService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		HubManager:      app.HubManager,
	})

	return &testServer{
		handler: router,
		storage: app.Storage.(*memory.Storage),
		auth:    app.AuthService,
	}
}

func (ts *testServer) request(method, path string, body any, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	ts.handler.ServeHTTP(rr, req)
	return rr
}

func TestHealthCheck(t *testing.T) {
	ts := newTestServer(t)

	rr := ts.request(http.MethodGet, "/api/v1/health", nil, "")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "ok")
}

func TestCreateGuestPlayer(t *testing.T) {
	ts := newTestServer(t)

	body := map[string]string{"display_name": "Alice"}
	rr := ts.request(http.MethodPost, "/api/v1/players/guest", body, "")

	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp response.AuthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "Alice", resp.Player.DisplayName)
	assert.True(t, resp.Player.IsGuest)
	assert.NotEmpty(t, resp.SessionToken)
}

func TestRegisterAndLogin(t *testing.T) {
	ts := newTestServer(t)

	// Register
	registerBody := map[string]string{
		"username":     "alice",
		"password":     "secret123",
		"display_name": "Alice",
	}
	rr := ts.request(http.MethodPost, "/api/v1/players/register", registerBody, "")
	assert.Equal(t, http.StatusCreated, rr.Code)

	var registerResp response.AuthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &registerResp)
	require.NoError(t, err)
	assert.False(t, registerResp.Player.IsGuest)

	// Login
	loginBody := map[string]string{
		"username": "alice",
		"password": "secret123",
	}
	rr = ts.request(http.MethodPost, "/api/v1/players/login", loginBody, "")
	assert.Equal(t, http.StatusOK, rr.Code)

	var loginResp response.AuthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	assert.Equal(t, registerResp.Player.ID, loginResp.Player.ID)
}

func TestGetMe(t *testing.T) {
	ts := newTestServer(t)

	// Create guest first
	body := map[string]string{"display_name": "Bob"}
	rr := ts.request(http.MethodPost, "/api/v1/players/guest", body, "")
	require.Equal(t, http.StatusCreated, rr.Code)

	var authResp response.AuthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &authResp)
	require.NoError(t, err)

	// Get me
	rr = ts.request(http.MethodGet, "/api/v1/players/me", nil, authResp.SessionToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	var meResp response.Player
	err = json.Unmarshal(rr.Body.Bytes(), &meResp)
	require.NoError(t, err)
	assert.Equal(t, "Bob", meResp.DisplayName)
}

func TestUnauthorizedWithoutToken(t *testing.T) {
	ts := newTestServer(t)

	// Try to get /me without token
	rr := ts.request(http.MethodGet, "/api/v1/players/me", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Try to create lobby without token
	rr = ts.request(http.MethodPost, "/api/v1/lobbies", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateAndJoinLobby(t *testing.T) {
	ts := newTestServer(t)

	// Create two players
	token1 := createGuestPlayer(t, ts, "Alice")
	token2 := createGuestPlayer(t, ts, "Bob")

	// Alice creates a lobby
	body := map[string]int{"grid_size": 5}
	rr := ts.request(http.MethodPost, "/api/v1/lobbies", body, token1)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var lobbyResp response.Lobby
	err := json.Unmarshal(rr.Body.Bytes(), &lobbyResp)
	require.NoError(t, err)

	assert.Equal(t, "waiting", lobbyResp.State)
	assert.Equal(t, 5, lobbyResp.Config.GridSize)
	assert.Len(t, lobbyResp.Members, 1)
	assert.True(t, lobbyResp.Members[0].IsHost)

	// Bob joins the lobby
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyResp.Code+"/join", nil, token2)
	assert.Equal(t, http.StatusOK, rr.Code)

	var joinResp response.Lobby
	err = json.Unmarshal(rr.Body.Bytes(), &joinResp)
	require.NoError(t, err)
	assert.Len(t, joinResp.Members, 2)
}

func TestLobbyHostActions(t *testing.T) {
	ts := newTestServer(t)

	token1 := createGuestPlayer(t, ts, "Alice")
	token2 := createGuestPlayer(t, ts, "Bob")

	// Create lobby
	lobbyCode := createLobby(t, ts, token1, 5)

	// Bob joins
	rr := ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/join", nil, token2)
	require.Equal(t, http.StatusOK, rr.Code)

	// Bob tries to update config (should fail - not host)
	body := map[string]int{"grid_size": 7}
	rr = ts.request(http.MethodPatch, "/api/v1/lobbies/"+lobbyCode+"/config", body, token2)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Alice updates config (should succeed)
	rr = ts.request(http.MethodPatch, "/api/v1/lobbies/"+lobbyCode+"/config", body, token1)
	assert.Equal(t, http.StatusOK, rr.Code)

	var configResp response.LobbyConfig
	err := json.Unmarshal(rr.Body.Bytes(), &configResp)
	require.NoError(t, err)
	assert.Equal(t, 7, configResp.GridSize)
}

func TestFullGameFlow(t *testing.T) {
	ts := newTestServer(t)

	// Create two players
	token1 := createGuestPlayer(t, ts, "Alice")
	token2 := createGuestPlayer(t, ts, "Bob")

	// Create lobby and have Bob join
	lobbyCode := createLobby(t, ts, token1, 3) // 3x3 = 9 turns

	rr := ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/join", nil, token2)
	require.Equal(t, http.StatusOK, rr.Code)

	// Start game
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token1)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var gameResp response.GameState
	err := json.Unmarshal(rr.Body.Bytes(), &gameResp)
	require.NoError(t, err)
	assert.Equal(t, "announcing", gameResp.State)
	assert.Len(t, gameResp.Players, 2)

	// Get game state
	rr = ts.request(http.MethodGet, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token1)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Play one turn - determine who is announcer
	announcer := gameResp.CurrentAnnouncer
	announcerToken := token1
	otherToken := token2
	if announcer != gameResp.Players[0] {
		announcerToken = token2
		otherToken = token1
	}

	// Announce letter
	announceBody := map[string]string{"letter": "A"}
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/game/announce", announceBody, announcerToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	var announceResp response.AnnounceResponse
	err = json.Unmarshal(rr.Body.Bytes(), &announceResp)
	require.NoError(t, err)
	assert.Equal(t, "placing", announceResp.State)
	assert.Equal(t, "A", announceResp.CurrentLetter)

	// Both players place
	placeBody := map[string]int{"row": 0, "col": 0}
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/game/place", placeBody, announcerToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	var placeResp response.PlaceResponse
	err = json.Unmarshal(rr.Body.Bytes(), &placeResp)
	require.NoError(t, err)
	assert.True(t, placeResp.Placed)
	assert.False(t, placeResp.TurnComplete) // Other player hasn't placed yet

	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/game/place", placeBody, otherToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	err = json.Unmarshal(rr.Body.Bytes(), &placeResp)
	require.NoError(t, err)
	assert.True(t, placeResp.TurnComplete) // All players placed
}

func TestAbandonGame(t *testing.T) {
	ts := newTestServer(t)

	token1 := createGuestPlayer(t, ts, "Alice")
	token2 := createGuestPlayer(t, ts, "Bob")

	lobbyCode := createLobby(t, ts, token1, 5)

	rr := ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/join", nil, token2)
	require.Equal(t, http.StatusOK, rr.Code)

	// Start game
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token1)
	require.Equal(t, http.StatusCreated, rr.Code)

	// Non-host tries to abandon (should fail)
	rr = ts.request(http.MethodDelete, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token2)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Host abandons
	rr = ts.request(http.MethodDelete, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token1)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify no game in progress
	rr = ts.request(http.MethodGet, "/api/v1/lobbies/"+lobbyCode+"/game", nil, token1)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestLeaveLobby(t *testing.T) {
	ts := newTestServer(t)

	token1 := createGuestPlayer(t, ts, "Alice")
	token2 := createGuestPlayer(t, ts, "Bob")

	lobbyCode := createLobby(t, ts, token1, 5)

	// Bob joins
	rr := ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/join", nil, token2)
	require.Equal(t, http.StatusOK, rr.Code)

	// Bob leaves
	rr = ts.request(http.MethodPost, "/api/v1/lobbies/"+lobbyCode+"/leave", nil, token2)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify Bob is gone
	rr = ts.request(http.MethodGet, "/api/v1/lobbies/"+lobbyCode, nil, token1)
	assert.Equal(t, http.StatusOK, rr.Code)

	var lobbyResp response.Lobby
	err := json.Unmarshal(rr.Body.Bytes(), &lobbyResp)
	require.NoError(t, err)
	assert.Len(t, lobbyResp.Members, 1)
}

// Helper functions

func createGuestPlayer(t *testing.T, ts *testServer, displayName string) string {
	t.Helper()

	body := map[string]string{"display_name": displayName}
	rr := ts.request(http.MethodPost, "/api/v1/players/guest", body, "")
	require.Equal(t, http.StatusCreated, rr.Code)

	var resp response.AuthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp.SessionToken
}

func createLobby(t *testing.T, ts *testServer, token string, gridSize int) string {
	t.Helper()

	body := map[string]int{"grid_size": gridSize}
	rr := ts.request(http.MethodPost, "/api/v1/lobbies", body, token)
	require.Equal(t, http.StatusCreated, rr.Code)

	var resp response.Lobby
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp.Code
}
