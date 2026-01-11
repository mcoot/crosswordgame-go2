package apierr

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
)

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps an APIError
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// Common error codes
const (
	CodeInvalidRequest      = "INVALID_REQUEST"
	CodeInvalidLetter       = "INVALID_LETTER"
	CodeInvalidPosition     = "INVALID_POSITION"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeNotHost             = "NOT_HOST"
	CodeNotYourTurn         = "NOT_YOUR_TURN"
	CodeAlreadyPlaced       = "ALREADY_PLACED"
	CodePlayerNotFound      = "PLAYER_NOT_FOUND"
	CodeLobbyNotFound       = "LOBBY_NOT_FOUND"
	CodeGameNotFound        = "GAME_NOT_FOUND"
	CodeAlreadyInLobby      = "ALREADY_IN_LOBBY"
	CodeNotInLobby          = "NOT_IN_LOBBY"
	CodeGameInProgress      = "GAME_IN_PROGRESS"
	CodeNoGameInProgress    = "NO_GAME_IN_PROGRESS"
	CodeCellOccupied        = "CELL_OCCUPIED"
	CodeInsufficientPlayers = "INSUFFICIENT_PLAYERS"
	CodeUsernameExists      = "USERNAME_EXISTS"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeInternalError       = "INTERNAL_ERROR"
)

// httpError combines an HTTP status code with an APIError
type httpError struct {
	status   int
	apiError APIError
}

// Error implements error interface
func (e *httpError) Error() string {
	return e.apiError.Message
}

// WriteError writes an error response to the response writer
func WriteError(w http.ResponseWriter, err error) {
	he := toHTTPError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(he.status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: he.apiError})
}

// toHTTPError converts an error to an httpError
func toHTTPError(err error) *httpError {
	// Check for specific error types
	var he *httpError
	if errors.As(err, &he) {
		return he
	}

	// Map model errors
	switch {
	case errors.Is(err, model.ErrPlayerNotFound):
		return &httpError{http.StatusNotFound, APIError{CodePlayerNotFound, "Player not found"}}
	case errors.Is(err, model.ErrLobbyNotFound):
		return &httpError{http.StatusNotFound, APIError{CodeLobbyNotFound, "Lobby not found"}}
	case errors.Is(err, model.ErrGameNotFound):
		return &httpError{http.StatusNotFound, APIError{CodeGameNotFound, "Game not found"}}
	case errors.Is(err, model.ErrAlreadyInLobby):
		return &httpError{http.StatusConflict, APIError{CodeAlreadyInLobby, "Already in this lobby"}}
	case errors.Is(err, model.ErrNotInLobby):
		return &httpError{http.StatusNotFound, APIError{CodeNotInLobby, "Not in this lobby"}}
	case errors.Is(err, model.ErrNotHost):
		return &httpError{http.StatusForbidden, APIError{CodeNotHost, "Only the host can perform this action"}}
	case errors.Is(err, model.ErrGameInProgress):
		return &httpError{http.StatusConflict, APIError{CodeGameInProgress, "Game is in progress"}}
	case errors.Is(err, model.ErrNoGameInProgress):
		return &httpError{http.StatusNotFound, APIError{CodeNoGameInProgress, "No game in progress"}}
	case errors.Is(err, model.ErrInsufficientPlayers):
		return &httpError{http.StatusConflict, APIError{CodeInsufficientPlayers, "Not enough players to start"}}
	case errors.Is(err, model.ErrNotPlayerTurn):
		return &httpError{http.StatusForbidden, APIError{CodeNotYourTurn, "Not your turn"}}
	case errors.Is(err, model.ErrInvalidLetter):
		return &httpError{http.StatusBadRequest, APIError{CodeInvalidLetter, "Letter must be A-Z"}}
	case errors.Is(err, model.ErrLetterNotAnnounced):
		return &httpError{http.StatusConflict, APIError{CodeNoGameInProgress, "No letter has been announced"}}
	case errors.Is(err, model.ErrAlreadyPlaced):
		return &httpError{http.StatusForbidden, APIError{CodeAlreadyPlaced, "Already placed this turn"}}
	case errors.Is(err, model.ErrInvalidPosition):
		return &httpError{http.StatusBadRequest, APIError{CodeInvalidPosition, "Invalid board position"}}
	case errors.Is(err, model.ErrCellOccupied):
		return &httpError{http.StatusConflict, APIError{CodeCellOccupied, "Cell is already occupied"}}

	// Map auth errors
	case errors.Is(err, auth.ErrInvalidCredentials):
		return &httpError{http.StatusUnauthorized, APIError{CodeInvalidCredentials, "Invalid username or password"}}
	case errors.Is(err, auth.ErrInvalidSession):
		return &httpError{http.StatusUnauthorized, APIError{CodeUnauthorized, "Invalid or expired session"}}
	case errors.Is(err, auth.ErrUsernameExists):
		return &httpError{http.StatusConflict, APIError{CodeUsernameExists, "Username already exists"}}

	default:
		return &httpError{http.StatusInternalServerError, APIError{CodeInternalError, "Internal server error"}}
	}
}

// NewInvalidRequestError creates an invalid request error
func NewInvalidRequestError(message string) error {
	return &httpError{http.StatusBadRequest, APIError{CodeInvalidRequest, message}}
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError() error {
	return &httpError{http.StatusUnauthorized, APIError{CodeUnauthorized, "Authentication required"}}
}

// NewInternalError creates an internal server error
func NewInternalError() error {
	return &httpError{http.StatusInternalServerError, APIError{CodeInternalError, "Internal server error"}}
}
