package middleware

import (
	"context"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
)

type contextKey string

const (
	playerContextKey contextKey = "player"
)

// GetPlayer retrieves the authenticated player from the request context
// Returns nil if no player is authenticated
func GetPlayer(ctx context.Context) *model.Player {
	player, _ := ctx.Value(playerContextKey).(*model.Player)
	return player
}

// Auth returns middleware that requires authentication
// Redirects to home page if not authenticated
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			player := getPlayerFromSession(r, authService)
			if player == nil {
				// Store original URL to redirect back after auth
				redirectURL := "/?next=" + r.URL.Path
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), playerContextKey, player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth returns middleware that attempts authentication but doesn't require it
// Sets player in context if authenticated, nil otherwise
func OptionalAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			player := getPlayerFromSession(r, authService)
			ctx := context.WithValue(r.Context(), playerContextKey, player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getPlayerFromSession(r *http.Request, authService *auth.Service) *model.Player {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	player, err := authService.GetPlayer(cookie.Value)
	if err != nil {
		return nil
	}

	return player
}
