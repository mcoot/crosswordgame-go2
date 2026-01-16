package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcoot/crosswordgame-go2/internal/api"
	"github.com/mcoot/crosswordgame-go2/internal/factory"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/web"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// cliRunner manages CLI binary execution
type cliRunner struct {
	binaryPath string
	serverURL  string
	tokenFile  string
}

func newCLIRunner(t *testing.T, serverURL string) *cliRunner {
	t.Helper()

	// Find project root (where go.mod is)
	projectRoot := findProjectRoot(t)

	// Build the CLI binary
	binaryPath := filepath.Join(projectRoot, "bin", "cwgame-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/cwgame")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build CLI: %s", string(output))

	// Create temp token file
	tokenFile := filepath.Join(t.TempDir(), "token")

	return &cliRunner{
		binaryPath: binaryPath,
		serverURL:  serverURL,
		tokenFile:  tokenFile,
	}
}

func (r *cliRunner) run(args ...string) (string, error) {
	fullArgs := append([]string{
		"--server", r.serverURL,
		"--token-file", r.tokenFile,
		"--output", "json",
	}, args...)

	cmd := exec.Command(r.binaryPath, fullArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (r *cliRunner) runWithToken(token string, args ...string) (string, error) {
	fullArgs := append([]string{
		"--server", r.serverURL,
		"--token", token,
		"--output", "json",
	}, args...)

	cmd := exec.Command(r.binaryPath, fullArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// testServer manages a real HTTP server for e2e tests
type testServer struct {
	server   *http.Server
	addr     string
	shutdown func()
}

func startTestServer(t *testing.T) *testServer {
	t.Helper()

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	// Create application
	projectRoot := findProjectRoot(t)
	app, err := factory.New(factory.Config{
		DictionaryPath: filepath.Join(projectRoot, "data/words.txt"),
	})
	require.NoError(t, err)

	// Load dictionary
	err = app.DictionaryService.LoadFromFile(context.Background(), filepath.Join(projectRoot, "data/words.txt"))
	require.NoError(t, err)

	// Create services
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authService := auth.New(app.Storage, app.Clock, auth.DefaultConfig(), logger)
	hubManager := sse.NewHubManager(logger)

	// Create routers
	apiRouter := api.NewRouter(api.RouterConfig{
		Logger:          logger,
		AuthService:     authService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		HubManager:      hubManager,
	})

	webRouter := web.NewRouter(web.RouterConfig{
		Logger:          logger,
		AuthService:     authService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		ScoringService:  app.ScoringService,
		HubManager:      hubManager,
		StaticDir:       filepath.Join(projectRoot, "internal/web/static"),
	})

	// Combine routers
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	// Wait for server to be ready
	serverURL := "http://" + addr
	waitForServer(t, serverURL+"/api/v1/health")

	return &testServer{
		server: server,
		addr:   serverURL,
		shutdown: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		},
	}
}

func waitForServer(t *testing.T, url string) {
	t.Helper()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("server did not become ready in time")
}

// Response types for JSON parsing
type authResponse struct {
	Player struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		IsGuest     bool   `json:"is_guest"`
	} `json:"player"`
	SessionToken string `json:"session_token"`
}

type lobbyResponse struct {
	Code   string `json:"code"`
	State  string `json:"state"`
	Config struct {
		GridSize int `json:"grid_size"`
	} `json:"config"`
	Members []struct {
		PlayerID    string `json:"player_id"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		IsHost      bool   `json:"is_host"`
	} `json:"members"`
	CurrentGame *string `json:"current_game"`
}

type gameStateResponse struct {
	ID               string          `json:"id"`
	State            string          `json:"state"`
	GridSize         int             `json:"grid_size"`
	Players          []string        `json:"players"`
	CurrentTurn      int             `json:"current_turn"`
	CurrentAnnouncer string          `json:"current_announcer"`
	CurrentLetter    *string         `json:"current_letter"`
	Placements       map[string]bool `json:"placements"`
	MyBoard          *boardResponse  `json:"my_board"`
}

type boardResponse struct {
	Cells [][]string `json:"cells"`
}

type announceResponse struct {
	State         string `json:"state"`
	CurrentLetter string `json:"current_letter"`
}

type placeResponse struct {
	Placed        bool    `json:"placed"`
	TurnComplete  bool    `json:"turn_complete"`
	GameComplete  bool    `json:"game_complete"`
	NextAnnouncer string  `json:"next_announcer"`
	Winner        *string `json:"winner"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type messageResponse struct {
	Message string `json:"message"`
}

// Tests

func TestCLI_HealthCheck(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	cli := newCLIRunner(t, ts.addr)

	output, err := cli.run("health")
	require.NoError(t, err, "output: %s", output)

	var resp healthResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.Equal(t, "ok", resp.Status)
}

func TestCLI_PlayerCommands(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	cli := newCLIRunner(t, ts.addr)

	// Create guest
	output, err := cli.run("player", "guest", "--name", "Alice")
	require.NoError(t, err, "output: %s", output)

	var authResp authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &authResp))
	assert.Equal(t, "Alice", authResp.Player.DisplayName)
	assert.True(t, authResp.Player.IsGuest)
	assert.NotEmpty(t, authResp.SessionToken)

	// Get me (token should be saved in token file)
	output, err = cli.run("player", "me")
	require.NoError(t, err, "output: %s", output)

	var player struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		IsGuest     bool   `json:"is_guest"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &player))
	assert.Equal(t, "Alice", player.DisplayName)
	assert.Equal(t, authResp.Player.ID, player.ID)
}

func TestCLI_LobbyCommands(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	cli := newCLIRunner(t, ts.addr)

	// Create guest
	output, err := cli.run("player", "guest", "--name", "Alice")
	require.NoError(t, err, "output: %s", output)

	var authResp authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &authResp))
	token := authResp.SessionToken

	// Create lobby
	output, err = cli.runWithToken(token, "lobby", "create", "--grid-size", "5")
	require.NoError(t, err, "output: %s", output)

	var lobbyResp lobbyResponse
	require.NoError(t, json.Unmarshal([]byte(output), &lobbyResp))
	assert.Equal(t, "waiting", lobbyResp.State)
	assert.Equal(t, 5, lobbyResp.Config.GridSize)
	assert.Len(t, lobbyResp.Members, 1)
	assert.True(t, lobbyResp.Members[0].IsHost)
	lobbyCode := lobbyResp.Code

	// Get lobby
	output, err = cli.runWithToken(token, "lobby", "get", lobbyCode)
	require.NoError(t, err, "output: %s", output)

	var getLobbyResp lobbyResponse
	require.NoError(t, json.Unmarshal([]byte(output), &getLobbyResp))
	assert.Equal(t, lobbyCode, getLobbyResp.Code)

	// Update config
	output, err = cli.runWithToken(token, "lobby", "config", lobbyCode, "--grid-size", "7")
	require.NoError(t, err, "output: %s", output)

	var configResp struct {
		GridSize int `json:"grid_size"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &configResp))
	assert.Equal(t, 7, configResp.GridSize)

	// Leave lobby
	output, err = cli.runWithToken(token, "lobby", "leave", lobbyCode)
	require.NoError(t, err, "output: %s", output)

	var msgResp messageResponse
	require.NoError(t, json.Unmarshal([]byte(output), &msgResp))
	assert.Contains(t, msgResp.Message, "Left lobby")
}

func TestCLI_FullGameFlow(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	// Create two CLI runners with separate token files
	cli1 := newCLIRunner(t, ts.addr)
	cli2 := &cliRunner{
		binaryPath: cli1.binaryPath,
		serverURL:  cli1.serverURL,
		tokenFile:  filepath.Join(t.TempDir(), "token2"),
	}

	// Create two players
	output, err := cli1.run("player", "guest", "--name", "Alice")
	require.NoError(t, err, "output: %s", output)
	var auth1 authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &auth1))
	token1 := auth1.SessionToken

	output, err = cli2.run("player", "guest", "--name", "Bob")
	require.NoError(t, err, "output: %s", output)
	var auth2 authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &auth2))
	token2 := auth2.SessionToken

	// Alice creates a lobby with 3x3 grid (9 turns)
	output, err = cli1.runWithToken(token1, "lobby", "create", "--grid-size", "3")
	require.NoError(t, err, "output: %s", output)
	var lobby lobbyResponse
	require.NoError(t, json.Unmarshal([]byte(output), &lobby))
	lobbyCode := lobby.Code
	t.Logf("Created lobby: %s", lobbyCode)

	// Bob joins the lobby
	output, err = cli2.runWithToken(token2, "lobby", "join", lobbyCode)
	require.NoError(t, err, "output: %s", output)
	require.NoError(t, json.Unmarshal([]byte(output), &lobby))
	assert.Len(t, lobby.Members, 2)
	t.Logf("Bob joined lobby")

	// Alice starts the game
	output, err = cli1.runWithToken(token1, "game", "start", lobbyCode)
	require.NoError(t, err, "output: %s", output)
	var gameState gameStateResponse
	require.NoError(t, json.Unmarshal([]byte(output), &gameState))
	assert.Equal(t, "announcing", gameState.State)
	assert.Len(t, gameState.Players, 2)
	t.Logf("Game started, announcer: %s", gameState.CurrentAnnouncer)

	// Play all 9 turns (3x3 grid)
	letters := []string{"C", "A", "T", "D", "O", "G", "S", "U", "N"}
	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 0}, {1, 1}, {1, 2},
		{2, 0}, {2, 1}, {2, 2},
	}

	for turn := 0; turn < 9; turn++ {
		// Get current game state
		output, err = cli1.runWithToken(token1, "game", "get", lobbyCode)
		require.NoError(t, err, "output: %s", output)
		require.NoError(t, json.Unmarshal([]byte(output), &gameState))

		// Determine announcer
		announcerToken := token1
		if gameState.CurrentAnnouncer == string(auth2.Player.ID) {
			announcerToken = token2
		}

		// Announce letter
		letter := letters[turn]
		output, err = cli1.runWithToken(announcerToken, "game", "announce", lobbyCode, letter)
		require.NoError(t, err, "turn %d announce: %s", turn, output)
		var announce announceResponse
		require.NoError(t, json.Unmarshal([]byte(output), &announce))
		assert.Equal(t, "placing", announce.State)
		assert.Equal(t, letter, announce.CurrentLetter)
		t.Logf("Turn %d: announced %s", turn, letter)

		// Both players place at same position
		row, col := positions[turn][0], positions[turn][1]

		output, err = cli1.runWithToken(token1, "game", "place", lobbyCode, fmt.Sprintf("%d", row), fmt.Sprintf("%d", col))
		require.NoError(t, err, "turn %d place1: %s", turn, output)
		var place1 placeResponse
		require.NoError(t, json.Unmarshal([]byte(output), &place1))
		assert.True(t, place1.Placed)
		assert.False(t, place1.TurnComplete) // Bob hasn't placed yet

		output, err = cli1.runWithToken(token2, "game", "place", lobbyCode, fmt.Sprintf("%d", row), fmt.Sprintf("%d", col))
		require.NoError(t, err, "turn %d place2: %s", turn, output)
		var place2 placeResponse
		require.NoError(t, json.Unmarshal([]byte(output), &place2))
		assert.True(t, place2.Placed)
		assert.True(t, place2.TurnComplete)
		t.Logf("Turn %d: both placed at (%d, %d)", turn, row, col)

		// Check if game complete
		if place2.GameComplete {
			t.Logf("Game complete! Winner: %v", place2.Winner)
			// After game completes, the lobby returns to waiting state with no game
			// Verify by checking the lobby
			output, err = cli1.runWithToken(token1, "lobby", "get", lobbyCode)
			require.NoError(t, err, "output: %s", output)
			require.NoError(t, json.Unmarshal([]byte(output), &lobby))
			assert.Equal(t, "waiting", lobby.State)
			assert.Nil(t, lobby.CurrentGame)
			t.Logf("Lobby returned to waiting state")
			return
		}
	}

	t.Fatal("Game should have completed after 9 turns")
}

func TestCLI_GameAbandon(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	cli1 := newCLIRunner(t, ts.addr)
	cli2 := &cliRunner{
		binaryPath: cli1.binaryPath,
		serverURL:  cli1.serverURL,
		tokenFile:  filepath.Join(t.TempDir(), "token2"),
	}

	// Create players
	output, err := cli1.run("player", "guest", "--name", "Alice")
	require.NoError(t, err)
	var auth1 authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &auth1))
	token1 := auth1.SessionToken

	output, err = cli2.run("player", "guest", "--name", "Bob")
	require.NoError(t, err)
	var auth2 authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &auth2))
	token2 := auth2.SessionToken

	// Create lobby and have Bob join
	output, err = cli1.runWithToken(token1, "lobby", "create")
	require.NoError(t, err)
	var lobby lobbyResponse
	require.NoError(t, json.Unmarshal([]byte(output), &lobby))
	lobbyCode := lobby.Code

	_, err = cli2.runWithToken(token2, "lobby", "join", lobbyCode)
	require.NoError(t, err)

	// Start game
	_, err = cli1.runWithToken(token1, "game", "start", lobbyCode)
	require.NoError(t, err)

	// Bob tries to abandon (should fail - not host)
	output, err = cli1.runWithToken(token2, "game", "abandon", lobbyCode)
	assert.Error(t, err, "non-host should not be able to abandon")
	assert.Contains(t, strings.ToLower(output), "not")

	// Alice abandons (should succeed)
	output, err = cli1.runWithToken(token1, "game", "abandon", lobbyCode)
	require.NoError(t, err, "output: %s", output)

	var msgResp messageResponse
	require.NoError(t, json.Unmarshal([]byte(output), &msgResp))
	assert.Equal(t, "Game abandoned", msgResp.Message)

	// Verify no game
	_, err = cli1.runWithToken(token1, "game", "get", lobbyCode)
	assert.Error(t, err, "should not find game after abandon")
}

func TestCLI_ErrorHandling(t *testing.T) {
	ts := startTestServer(t)
	defer ts.shutdown()

	cli := newCLIRunner(t, ts.addr)

	// Get player without auth
	output, err := cli.run("player", "me")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(output), "unauthorized")

	// Get non-existent lobby
	output, err = cli.run("player", "guest", "--name", "Alice")
	require.NoError(t, err)
	var auth authResponse
	require.NoError(t, json.Unmarshal([]byte(output), &auth))

	output, err = cli.runWithToken(auth.SessionToken, "lobby", "get", "INVALID")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(output), "not found")
}
