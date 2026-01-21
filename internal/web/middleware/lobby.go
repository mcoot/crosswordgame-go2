package middleware

import (
	"context"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
)

const (
	activeLobbyCodeContextKey contextKey = "activeLobbyCode"
)

// GetActiveLobbyCode retrieves the active lobby code from the request context
// Returns empty string if no active lobby
func GetActiveLobbyCode(ctx context.Context) model.LobbyCode {
	code, _ := ctx.Value(activeLobbyCodeContextKey).(model.LobbyCode)
	return code
}

// ActiveLobby returns middleware that queries for the player's active lobby
// and adds it to the context. Requires auth middleware to be applied first.
func ActiveLobby(lobbyController *lobby.Controller) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			player := GetPlayer(r.Context())
			ctx := r.Context()

			if player != nil {
				lobbyCode, err := lobbyController.GetActiveLobbyCode(r.Context(), player.ID)
				if err == nil && lobbyCode != "" {
					ctx = context.WithValue(ctx, activeLobbyCodeContextKey, lobbyCode)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
