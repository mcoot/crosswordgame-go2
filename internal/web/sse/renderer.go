package sse

import (
	"bytes"
	"context"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/components"
)

// Renderer converts model events to HTML fragments for SSE
type Renderer struct{}

// NewRenderer creates a new Renderer
func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderMemberList renders the member list component as HTML
func (r *Renderer) RenderMemberList(ctx context.Context, lobby *model.Lobby, currentPlayerID model.PlayerID, isHost bool) (string, error) {
	var buf bytes.Buffer
	err := components.MemberList(lobby, currentPlayerID, isHost).Render(ctx, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderLobbyControls renders the lobby controls component as HTML
func (r *Renderer) RenderLobbyControls(ctx context.Context, lobby *model.Lobby) (string, error) {
	var buf bytes.Buffer
	err := components.LobbyControls(lobby).Render(ctx, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderLobbyConfig renders the lobby config component as HTML
func (r *Renderer) RenderLobbyConfig(ctx context.Context, lobby *model.Lobby) (string, error) {
	var buf bytes.Buffer
	err := components.LobbyConfig(lobby).Render(ctx, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WrapForOOBSwap wraps HTML in a div with hx-swap-oob for out-of-band swaps
func WrapForOOBSwap(id, html string) string {
	return `<div id="` + id + `" hx-swap-oob="true">` + html + `</div>`
}

// EventData represents SSE event data
type EventData struct {
	EventName string
	HTML      string
}

// RenderLobbyEvent converts a lobby event to SSE data
// Returns the event name and HTML fragment to broadcast
func (r *Renderer) RenderLobbyEvent(ctx context.Context, event model.Event, lobby *model.Lobby, viewerPlayerID model.PlayerID) ([]EventData, error) {
	var events []EventData

	// Determine if viewer is host
	host := lobby.GetHost()
	isHost := host != nil && host.Player.ID == viewerPlayerID

	switch event.Type {
	case model.EventPlayerJoined, model.EventPlayerLeft, model.EventRoleChanged:
		// Re-render member list
		html, err := r.RenderMemberList(ctx, lobby, viewerPlayerID, isHost)
		if err != nil {
			return nil, err
		}
		events = append(events, EventData{
			EventName: "member-update",
			HTML:      html,
		})

	case model.EventHostChanged:
		// Re-render member list and controls
		memberHTML, err := r.RenderMemberList(ctx, lobby, viewerPlayerID, isHost)
		if err != nil {
			return nil, err
		}
		events = append(events, EventData{
			EventName: "member-update",
			HTML:      memberHTML,
		})

		if isHost {
			controlsHTML, err := r.RenderLobbyControls(ctx, lobby)
			if err != nil {
				return nil, err
			}
			events = append(events, EventData{
				EventName: "controls-update",
				HTML:      WrapForOOBSwap("lobby-controls", controlsHTML),
			})
		}

	case model.EventGameStarted:
		// Signal to redirect to game page
		events = append(events, EventData{
			EventName: "game-started",
			HTML:      `<script>window.location.href = window.location.href + "/game";</script>`,
		})

	case model.EventGameEnded:
		// Signal game ended, refresh lobby
		events = append(events, EventData{
			EventName: "game-ended",
			HTML:      `<script>window.location.reload();</script>`,
		})
	}

	return events, nil
}
