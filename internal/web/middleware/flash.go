package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
)

const (
	flashCookieName = "flash"
	flashContextKey = contextKey("flash")
)

// GetFlash retrieves the flash message from the request context
// Returns nil if no flash message is set
func GetFlash(ctx context.Context) *layout.FlashMessage {
	flash, _ := ctx.Value(flashContextKey).(*layout.FlashMessage)
	return flash
}

// SetFlash sets a flash message to be displayed on the next request
func SetFlash(w http.ResponseWriter, flashType, message string) {
	// Encode as type:message
	value := flashType + ":" + message
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   60, // 1 minute expiry
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Flash returns middleware that reads and clears flash messages
func Flash() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var flash *layout.FlashMessage

			cookie, err := r.Cookie(flashCookieName)
			if err == nil && cookie.Value != "" {
				// Parse flash message
				flash = parseFlash(cookie.Value)

				// Clear the cookie
				http.SetCookie(w, &http.Cookie{
					Name:     flashCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					Expires:  time.Unix(0, 0),
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			ctx := context.WithValue(r.Context(), flashContextKey, flash)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseFlash(value string) *layout.FlashMessage {
	// Find first colon separator
	for i := 0; i < len(value); i++ {
		if value[i] == ':' {
			return &layout.FlashMessage{
				Type:    value[:i],
				Message: value[i+1:],
			}
		}
	}
	// If no colon, treat entire value as message with default type
	return &layout.FlashMessage{
		Type:    "info",
		Message: value,
	}
}
