package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/mcoot/crosswordgame-go2/internal/api/apierr"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
)

type contextKey string

const (
	playerContextKey  contextKey = "player"
	sessionContextKey contextKey = "session"
)

// Auth creates authentication middleware
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				apierr.WriteError(w, apierr.NewUnauthorizedError())
				return
			}

			session, err := authService.ValidateSession(token)
			if err != nil {
				apierr.WriteError(w, err)
				return
			}

			// Add session and player to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, sessionContextKey, session)
			ctx = context.WithValue(ctx, playerContextKey, &session.Player)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth extracts session if present but doesn't require it
func OptionalAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				if session, err := authService.ValidateSession(token); err == nil {
					ctx := r.Context()
					ctx = context.WithValue(ctx, sessionContextKey, session)
					ctx = context.WithValue(ctx, playerContextKey, &session.Player)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extracts the session token from the request
func extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fall back to cookie
	cookie, err := r.Cookie("session")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// GetPlayer returns the authenticated player from the request context
func GetPlayer(ctx context.Context) *model.Player {
	player, _ := ctx.Value(playerContextKey).(*model.Player)
	return player
}

// GetSession returns the session from the request context
func GetSession(ctx context.Context) *auth.Session {
	session, _ := ctx.Value(sessionContextKey).(*auth.Session)
	return session
}

// MustGetPlayer returns the authenticated player or panics
func MustGetPlayer(ctx context.Context) *model.Player {
	player := GetPlayer(ctx)
	if player == nil {
		panic("no player in context - auth middleware not applied?")
	}
	return player
}
