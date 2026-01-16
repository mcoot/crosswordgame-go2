package redis

import (
	"fmt"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// Key prefix for all game-related data
const keyPrefix = "cwgame"

// Key generation functions for each entity type

// playerKey returns the Redis key for a Player
func playerKey(id model.PlayerID) string {
	return fmt.Sprintf("%s:player:%s", keyPrefix, id)
}

// registeredPlayerKey returns the Redis key for a RegisteredPlayer
func registeredPlayerKey(playerID model.PlayerID) string {
	return fmt.Sprintf("%s:registered_player:%s", keyPrefix, playerID)
}

// usernameIndexKey returns the Redis key for the username -> player_id index
func usernameIndexKey(username string) string {
	return fmt.Sprintf("%s:idx:username:%s", keyPrefix, username)
}

// lobbyKey returns the Redis key for a Lobby
func lobbyKey(code model.LobbyCode) string {
	return fmt.Sprintf("%s:lobby:%s", keyPrefix, code)
}

// gameKey returns the Redis key for a Game
func gameKey(id model.GameID) string {
	return fmt.Sprintf("%s:game:%s", keyPrefix, id)
}

// boardKey returns the Redis key for a Board
func boardKey(gameID model.GameID, playerID model.PlayerID) string {
	return fmt.Sprintf("%s:board:%s:%s", keyPrefix, gameID, playerID)
}

// boardsForGameIndexKey returns the Redis key for the SET of boards for a game
func boardsForGameIndexKey(gameID model.GameID) string {
	return fmt.Sprintf("%s:idx:boards_for_game:%s", keyPrefix, gameID)
}

// dictionaryKey returns the Redis key for the dictionary word set
func dictionaryKey() string {
	return fmt.Sprintf("%s:dictionary", keyPrefix)
}
