package model

import "errors"

// Common errors used across the application
var (
	// Player errors
	ErrPlayerNotFound = errors.New("player not found")

	// Lobby errors
	ErrLobbyNotFound      = errors.New("lobby not found")
	ErrLobbyFull          = errors.New("lobby is full")
	ErrAlreadyInLobby     = errors.New("player is already in lobby")
	ErrNotInLobby         = errors.New("player is not in lobby")
	ErrNotHost            = errors.New("player is not the host")
	ErrGameInProgress     = errors.New("game is in progress")
	ErrNoGameInProgress   = errors.New("no game in progress")
	ErrInsufficientPlayers = errors.New("insufficient players to start game")

	// Game errors
	ErrGameNotFound       = errors.New("game not found")
	ErrNotPlayerTurn      = errors.New("not this player's turn")
	ErrInvalidLetter      = errors.New("invalid letter")
	ErrLetterNotAnnounced = errors.New("no letter has been announced")
	ErrAlreadyPlaced      = errors.New("player has already placed this turn")
	ErrInvalidPosition    = errors.New("invalid board position")
	ErrCellOccupied       = errors.New("cell is already occupied")
	ErrGameComplete       = errors.New("game is already complete")
	ErrGameAbandoned      = errors.New("game has been abandoned")

	// Board errors
	ErrBoardNotFound = errors.New("board not found")

	// Dictionary errors
	ErrDictionaryNotLoaded = errors.New("dictionary not loaded")
)
