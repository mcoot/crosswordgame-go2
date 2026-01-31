package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

const (
	// PlayerIDAlphabet is the character set for generating bot player IDs
	PlayerIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	// PlayerIDLength is the length of generated bot player IDs
	PlayerIDLength = 16
	// MaxBotIterations is a safety limit for the ProcessBotActions loop
	MaxBotIterations = 1000
)

// BotActionType represents the type of action a bot took
type BotActionType string

const (
	ActionAnnounce     BotActionType = "announce"
	ActionPlace        BotActionType = "place"
	ActionTurnComplete BotActionType = "turn_complete"
	ActionGameComplete BotActionType = "game_complete"
)

// BotAction represents a single action taken by a bot during ProcessBotActions
type BotAction struct {
	Type     BotActionType
	PlayerID model.PlayerID
	Letter   rune
	Position model.Position
}

// Service manages bot players in the game
type Service struct {
	storage         storage.Storage
	lobbyController *lobby.Controller
	gameController  *game.Controller
	boardService    *board.Service
	strategies      map[string]Strategy
	clock           clock.Clock
	random          random.Random
	logger          *slog.Logger
}

// NewService creates a new bot Service
func NewService(
	store storage.Storage,
	lobbyController *lobby.Controller,
	gameController *game.Controller,
	boardService *board.Service,
	strategies map[string]Strategy,
	clk clock.Clock,
	rnd random.Random,
	logger *slog.Logger,
) *Service {
	return &Service{
		storage:         store,
		lobbyController: lobbyController,
		gameController:  gameController,
		boardService:    boardService,
		strategies:      strategies,
		clock:           clk,
		random:          rnd,
		logger:          logger.With(slog.String("component", "bot-service")),
	}
}

// CreateBotPlayer creates a new bot player and saves it to storage
func (s *Service) CreateBotPlayer(ctx context.Context, displayName string, strategy string) (*model.Player, error) {
	player := &model.Player{
		ID:          model.PlayerID("bot-" + s.random.String(PlayerIDLength, PlayerIDAlphabet)),
		DisplayName: displayName,
		IsGuest:     true,
		IsBot:       true,
		BotStrategy: strategy,
		CreatedAt:   s.clock.Now(),
	}

	if err := s.storage.SavePlayer(ctx, player); err != nil {
		return nil, err
	}

	return player, nil
}

// AddBotToLobby creates a bot player and adds it to the lobby
// Only the lobby host can add bots, and only while in waiting state
func (s *Service) AddBotToLobby(ctx context.Context, code model.LobbyCode, requestingPlayerID model.PlayerID, strategy string) (*model.Player, error) {
	// Validate strategy
	if _, ok := s.strategies[strategy]; !ok {
		return nil, fmt.Errorf("unknown bot strategy: %s", strategy)
	}

	lob, err := s.lobbyController.GetLobby(ctx, code)
	if err != nil {
		return nil, err
	}

	// Verify requester is host
	host := lob.GetHost()
	if host == nil || host.Player.ID != requestingPlayerID {
		return nil, model.ErrNotHost
	}

	// Cannot add bots during a game
	if lob.State == model.LobbyStateInGame {
		return nil, model.ErrGameInProgress
	}

	// Count existing bots for naming
	botCount := 0
	for _, m := range lob.Members {
		if m.Player.IsBot {
			botCount++
		}
	}

	displayName := fmt.Sprintf("Bot %d", botCount+1)
	bot, err := s.CreateBotPlayer(ctx, displayName, strategy)
	if err != nil {
		return nil, err
	}

	if err := s.lobbyController.JoinLobby(ctx, code, *bot); err != nil {
		return nil, err
	}

	s.logger.Info("bot added to lobby",
		slog.String("lobby_code", string(code)),
		slog.String("bot_id", string(bot.ID)),
		slog.String("bot_name", displayName),
	)

	return bot, nil
}

// RemoveBotFromLobby removes a bot player from the lobby
// Only the lobby host can remove bots, and only while in waiting state
func (s *Service) RemoveBotFromLobby(ctx context.Context, code model.LobbyCode, requestingPlayerID model.PlayerID, botPlayerID model.PlayerID) error {
	lob, err := s.lobbyController.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Verify requester is host
	host := lob.GetHost()
	if host == nil || host.Player.ID != requestingPlayerID {
		return model.ErrNotHost
	}

	// Cannot remove bots during a game
	if lob.State == model.LobbyStateInGame {
		return model.ErrGameInProgress
	}

	// Verify target is a bot
	member := lob.GetMember(botPlayerID)
	if member == nil {
		return model.ErrNotInLobby
	}
	if !member.Player.IsBot {
		return model.ErrNotBot
	}

	return s.lobbyController.LeaveLobby(ctx, code, botPlayerID)
}

// ProcessBotActions executes bot actions in a cascading loop
// It returns all actions taken so handlers can broadcast SSE updates
func (s *Service) ProcessBotActions(ctx context.Context, gameID model.GameID) ([]BotAction, error) {
	var actions []BotAction

	for range MaxBotIterations {
		g, err := s.gameController.GetGame(ctx, gameID)
		if err != nil {
			return actions, err
		}

		// Stop if game is finished
		if g.State == model.GameStateScoring || g.State == model.GameStateAbandoned {
			if g.State == model.GameStateScoring && len(actions) > 0 {
				actions = append(actions, BotAction{Type: ActionGameComplete})
			}
			break
		}

		if g.State == model.GameStateAnnouncing {
			announcer := g.CurrentAnnouncer()
			announcerPlayer, err := s.storage.GetPlayer(ctx, announcer)
			if err != nil {
				return actions, err
			}

			if !announcerPlayer.IsBot {
				break // Human's turn to announce
			}

			botStrategy := s.strategyForPlayer(announcerPlayer)
			letter := botStrategy.ChooseLetter(g)
			if err := s.gameController.AnnounceLetter(ctx, gameID, announcer, letter); err != nil {
				return actions, err
			}

			actions = append(actions, BotAction{
				Type:     ActionAnnounce,
				PlayerID: announcer,
				Letter:   letter,
			})
			continue
		}

		if g.State == model.GameStatePlacing {
			anyBotPlaced := false
			for _, pid := range g.Players {
				if g.Placements[pid] {
					continue // Already placed
				}

				player, err := s.storage.GetPlayer(ctx, pid)
				if err != nil {
					return actions, err
				}
				if !player.IsBot {
					continue // Human player
				}

				playerBoard, err := s.boardService.GetBoard(ctx, gameID, pid)
				if err != nil {
					return actions, err
				}

				botStrategy := s.strategyForPlayer(player)
				pos := botStrategy.ChoosePosition(g, playerBoard)
				if err := s.gameController.PlaceLetter(ctx, gameID, pid, pos); err != nil {
					return actions, err
				}

				actions = append(actions, BotAction{
					Type:     ActionPlace,
					PlayerID: pid,
					Position: pos,
				})
				anyBotPlaced = true
			}

			if !anyBotPlaced {
				break // Only humans left to place
			}

			// Re-read game to check if turn advanced
			g, err = s.gameController.GetGame(ctx, gameID)
			if err != nil {
				return actions, err
			}

			if g.State == model.GameStateScoring {
				actions = append(actions, BotAction{Type: ActionGameComplete})
				break
			}
			if g.State == model.GameStateAnnouncing {
				actions = append(actions, BotAction{Type: ActionTurnComplete})
			}
			continue
		}

		break // Unknown state
	}

	return actions, nil
}

// strategyForPlayer returns the strategy for a bot player, falling back to
// the first registered strategy if the player's strategy is not found
func (s *Service) strategyForPlayer(player *model.Player) Strategy {
	if st, ok := s.strategies[player.BotStrategy]; ok {
		return st
	}
	// Fallback: use first available strategy
	for _, st := range s.strategies {
		return st
	}
	return nil
}
