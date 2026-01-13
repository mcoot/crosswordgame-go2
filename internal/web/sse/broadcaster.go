package sse

import (
	"bytes"
	"context"
	"log/slog"
	"strconv"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/components"
)

// Broadcaster handles broadcasting updates to SSE clients
type Broadcaster struct {
	hubManager *HubManager
	logger     *slog.Logger
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster(hubManager *HubManager, logger *slog.Logger) *Broadcaster {
	return &Broadcaster{
		hubManager: hubManager,
		logger:     logger.With(slog.String("component", "sse-broadcaster")),
	}
}

// BroadcastMemberListUpdate broadcasts an updated member list to all lobby clients
func (b *Broadcaster) BroadcastMemberListUpdate(ctx context.Context, lobby *model.Lobby) {
	hub := b.hubManager.GetHub(lobby.Code)
	if hub == nil {
		return
	}

	// Render member list (we use empty player ID since we show all members the same)
	var buf bytes.Buffer
	err := components.MemberList(lobby, "", false).Render(ctx, &buf)
	if err != nil {
		b.logger.Error("sse failed to render member list",
			slog.String("lobby", string(lobby.Code)),
			slog.Any("error", err))
		return
	}

	// Wrap with OOB swap
	html := WrapForOOBSwap("member-list", buf.String())
	hub.BroadcastEvent("member-update", html)
}

// BroadcastLobbyControlsUpdate broadcasts updated lobby controls
func (b *Broadcaster) BroadcastLobbyControlsUpdate(ctx context.Context, lobby *model.Lobby) {
	hub := b.hubManager.GetHub(lobby.Code)
	if hub == nil {
		return
	}

	var buf bytes.Buffer
	err := components.LobbyControls(lobby).Render(ctx, &buf)
	if err != nil {
		b.logger.Error("sse failed to render lobby controls",
			slog.String("lobby", string(lobby.Code)),
			slog.Any("error", err))
		return
	}

	html := WrapForOOBSwap("lobby-controls", buf.String())
	hub.BroadcastEvent("controls-update", html)
}

// BroadcastGameStarted broadcasts that a game has started
// HTMX will trigger a fetch to the game page via hx-trigger="sse:game-started"
func (b *Broadcaster) BroadcastGameStarted(lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Just send any data to trigger the event - HTMX handles the navigation
	hub.BroadcastEvent("game-started", "started")
}

// BroadcastGameStatus broadcasts an updated game status
func (b *Broadcaster) BroadcastGameStatus(ctx context.Context, game *model.Game, lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// For game status, we broadcast a simple update that triggers a refresh
	// This is simpler than trying to render personalized views for each player
	var buf bytes.Buffer
	// We pass isAnnouncer=false and hasPlaced=false - clients will refresh to get accurate state
	err := components.GameStatus(game, false, false).Render(ctx, &buf)
	if err != nil {
		b.logger.Error("sse failed to render game status",
			slog.String("lobby", string(lobbyCode)),
			slog.Any("error", err))
		return
	}

	html := WrapForOOBSwap("game-status", buf.String())
	hub.BroadcastEvent("game-update", html)
}

// BroadcastLetterAnnounced broadcasts that a letter has been announced
// HTMX will trigger a page fetch via hx-trigger="sse:letter-announced"
func (b *Broadcaster) BroadcastLetterAnnounced(ctx context.Context, game *model.Game, lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send letter as data - HTMX will fetch the page to get full personalized state
	hub.BroadcastEvent("letter-announced", string(game.CurrentLetter))
}

// BroadcastPlacementUpdate broadcasts that a player has placed their letter
func (b *Broadcaster) BroadcastPlacementUpdate(ctx context.Context, game *model.Game, lobbyCode model.LobbyCode, playerID model.PlayerID) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Count how many have placed
	placedCount := 0
	for _, placed := range game.Placements {
		if placed {
			placedCount++
		}
	}
	totalPlayers := len(game.Players)

	html := `<div id="placement-status" hx-swap-oob="true" class="text-muted">
		` + strconv.Itoa(placedCount) + `/` + strconv.Itoa(totalPlayers) + ` players have placed
	</div>`

	hub.BroadcastEvent("placement-update", html)
}

// BroadcastTurnComplete broadcasts that all players have placed and a new turn is starting
// HTMX will trigger a page fetch via hx-trigger="sse:turn-complete"
func (b *Broadcaster) BroadcastTurnComplete(ctx context.Context, game *model.Game, lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send turn number as data - HTMX will fetch the page
	hub.BroadcastEvent("turn-complete", strconv.Itoa(game.CurrentTurn))
}

// BroadcastGameComplete broadcasts that the game is complete
// HTMX will trigger a page fetch via hx-trigger="sse:game-complete"
func (b *Broadcaster) BroadcastGameComplete(lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send simple signal - HTMX will fetch the page
	hub.BroadcastEvent("game-complete", "complete")
}

// BroadcastGameAbandoned broadcasts that the game has been abandoned
// HTMX will trigger a fetch to the lobby page via hx-trigger="sse:game-abandoned"
func (b *Broadcaster) BroadcastGameAbandoned(lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send simple signal - HTMX will fetch the lobby page
	hub.BroadcastEvent("game-abandoned", "abandoned")
}

// BroadcastRefresh tells all clients to refresh the page
// HTMX will trigger a page fetch via hx-trigger="sse:refresh"
func (b *Broadcaster) BroadcastRefresh(lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send simple signal - HTMX will fetch the page
	hub.BroadcastEvent("refresh", "refresh")
}

// BroadcastGameDismissed broadcasts that the game scores have been dismissed
// HTMX will trigger a fetch to the lobby page via hx-trigger="sse:game-dismissed"
func (b *Broadcaster) BroadcastGameDismissed(lobbyCode model.LobbyCode) {
	hub := b.hubManager.GetHub(lobbyCode)
	if hub == nil {
		return
	}

	// Send simple signal - HTMX will fetch the lobby page
	hub.BroadcastEvent("game-dismissed", "dismissed")
}
