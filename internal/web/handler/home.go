package handler

import (
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/pages"
)

// HomeHandler handles the home page
type HomeHandler struct{}

// NewHomeHandler creates a new HomeHandler
func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

// Home renders the home page
func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	flash := middleware.GetFlash(r.Context())
	next := r.URL.Query().Get("next")

	data := pages.HomeData{
		PageData: layout.PageData{
			Title:  "Home",
			Player: player,
			Flash:  flash,
		},
		Next: next,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Home(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
