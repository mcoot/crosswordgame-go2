package handler

import (
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/api/apierr"
)

// Re-export from apierr for convenience
type APIError = apierr.APIError
type ErrorResponse = apierr.ErrorResponse

// Re-export error codes
const (
	CodeInvalidRequest      = apierr.CodeInvalidRequest
	CodeInvalidLetter       = apierr.CodeInvalidLetter
	CodeInvalidPosition     = apierr.CodeInvalidPosition
	CodeUnauthorized        = apierr.CodeUnauthorized
	CodeNotHost             = apierr.CodeNotHost
	CodeNotYourTurn         = apierr.CodeNotYourTurn
	CodeAlreadyPlaced       = apierr.CodeAlreadyPlaced
	CodePlayerNotFound      = apierr.CodePlayerNotFound
	CodeLobbyNotFound       = apierr.CodeLobbyNotFound
	CodeGameNotFound        = apierr.CodeGameNotFound
	CodeAlreadyInLobby      = apierr.CodeAlreadyInLobby
	CodeNotInLobby          = apierr.CodeNotInLobby
	CodeGameInProgress      = apierr.CodeGameInProgress
	CodeNoGameInProgress    = apierr.CodeNoGameInProgress
	CodeCellOccupied        = apierr.CodeCellOccupied
	CodeInsufficientPlayers = apierr.CodeInsufficientPlayers
	CodeUsernameExists      = apierr.CodeUsernameExists
	CodeInvalidCredentials  = apierr.CodeInvalidCredentials
	CodeInternalError       = apierr.CodeInternalError
)

// WriteError writes an error response to the response writer
func WriteError(w http.ResponseWriter, err error) {
	apierr.WriteError(w, err)
}

// NewInvalidRequestError creates an invalid request error
func NewInvalidRequestError(message string) error {
	return apierr.NewInvalidRequestError(message)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError() error {
	return apierr.NewUnauthorizedError()
}

// NewInternalError creates an internal server error
func NewInternalError() error {
	return apierr.NewInternalError()
}
