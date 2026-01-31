package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/mcoot/crosswordgame-go2/internal/testutil"
)

type ServiceSuite struct {
	suite.Suite
	store      *memory.Storage
	mockClock  *mocks.MockClock
	mockRandom *mocks.MockRandom

	boardService    *board.Service
	gameController  *game.Controller
	lobbyController *lobby.Controller
	botService      *bot.Service

	ctx context.Context
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.store = memory.New()
	s.mockClock = mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	s.mockRandom = mocks.NewMockRandom()
	logger := testutil.NopLogger()
	s.ctx = context.Background()

	dictService := dictionary.New(s.store, logger)
	s.boardService = board.New(s.store, logger)
	scoringService := scoring.New(dictService)
	s.gameController = game.NewController(s.store, s.boardService, scoringService, s.mockClock, s.mockRandom, logger)
	s.lobbyController = lobby.NewController(s.store, s.gameController, s.mockClock, s.mockRandom, logger)

	strategy := bot.NewRandomStrategy(s.mockRandom)
	s.botService = bot.NewService(s.store, s.lobbyController, s.gameController, s.boardService, strategy, s.mockClock, s.mockRandom, logger)
}

func (s *ServiceSuite) createPlayer(id, name string) model.Player {
	p := model.Player{
		ID:          model.PlayerID(id),
		DisplayName: name,
		IsGuest:     true,
		CreatedAt:   s.mockClock.Now(),
	}
	_ = s.store.SavePlayer(s.ctx, &p)
	return p
}

func (s *ServiceSuite) TestCreateBotPlayer() {
	s.mockRandom.QueueString("abcdefghijklmnop")

	player, err := s.botService.CreateBotPlayer(s.ctx, "Bot 1")
	s.Require().NoError(err)

	s.Equal("Bot 1", player.DisplayName)
	s.True(player.IsBot)
	s.True(player.IsGuest)
	s.Equal(model.PlayerID("bot-abcdefghijklmnop"), player.ID)

	// Verify saved to storage
	retrieved, err := s.store.GetPlayer(s.ctx, player.ID)
	s.Require().NoError(err)
	s.True(retrieved.IsBot)
}

func (s *ServiceSuite) TestAddBotToLobby() {
	s.mockRandom.QueueString("LOBBY1") // lobby code
	host := s.createPlayer("host", "Host")
	lob, err := s.lobbyController.CreateLobby(s.ctx, host)
	s.Require().NoError(err)

	s.mockRandom.QueueString("abcdefghijklmnop") // bot player ID
	botPlayer, err := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)
	s.Require().NoError(err)

	s.Equal("Bot 1", botPlayer.DisplayName)
	s.True(botPlayer.IsBot)

	// Verify bot is in lobby
	updatedLobby, _ := s.lobbyController.GetLobby(s.ctx, lob.Code)
	s.Len(updatedLobby.Members, 2)
	s.Equal(botPlayer.ID, updatedLobby.Members[1].Player.ID)
	s.True(updatedLobby.Members[1].Player.IsBot)
}

func (s *ServiceSuite) TestAddBotToLobby_NotHost() {
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	nonHost := s.createPlayer("other", "Other")
	_ = s.lobbyController.JoinLobby(s.ctx, lob.Code, nonHost)

	_, err := s.botService.AddBotToLobby(s.ctx, lob.Code, nonHost.ID)
	s.ErrorIs(err, model.ErrNotHost)
}

func (s *ServiceSuite) TestAddBotToLobby_GameInProgress() {
	s.mockRandom.QueueString("LOBBY1", "GAME01")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)
	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	_, _ = s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	_, err := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)
	s.ErrorIs(err, model.ErrGameInProgress)
}

func (s *ServiceSuite) TestAddBotToLobby_SequentialNaming() {
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("bot1botid_abcdef")
	bot1, err := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)
	s.Require().NoError(err)
	s.Equal("Bot 1", bot1.DisplayName)

	s.mockRandom.QueueString("bot2botid_abcdef")
	bot2, err := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)
	s.Require().NoError(err)
	s.Equal("Bot 2", bot2.DisplayName)
}

func (s *ServiceSuite) TestRemoveBotFromLobby() {
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("abcdefghijklmnop")
	botPlayer, _ := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	err := s.botService.RemoveBotFromLobby(s.ctx, lob.Code, host.ID, botPlayer.ID)
	s.Require().NoError(err)

	updatedLobby, _ := s.lobbyController.GetLobby(s.ctx, lob.Code)
	s.Len(updatedLobby.Members, 1)
}

func (s *ServiceSuite) TestRemoveBotFromLobby_NotBot() {
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	human := s.createPlayer("human", "Human")
	_ = s.lobbyController.JoinLobby(s.ctx, lob.Code, human)

	err := s.botService.RemoveBotFromLobby(s.ctx, lob.Code, host.ID, human.ID)
	s.ErrorIs(err, model.ErrNotBot)
}

func (s *ServiceSuite) TestProcessBotActions_BotAnnounces() {
	s.mockRandom.QueueString("LOBBY1", "GAME01")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("abcdefghijklmnop")
	botPlayer, _ := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	g, _ := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	// Determine who is first announcer
	if g.CurrentAnnouncer() == botPlayer.ID {
		// Bot is first announcer - queue random values for letter choice
		s.mockRandom.QueueIntn(2) // letter 'C'
		actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
		s.Require().NoError(err)
		s.Require().NotEmpty(actions)
		s.Equal(bot.ActionAnnounce, actions[0].Type)
		s.Equal(botPlayer.ID, actions[0].PlayerID)
		s.Equal('C', actions[0].Letter)
	} else {
		// Host is first announcer - bot should not act
		actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
		s.Require().NoError(err)
		s.Empty(actions)
	}
}

func (s *ServiceSuite) TestProcessBotActions_BotPlaces() {
	s.mockRandom.QueueString("LOBBY1", "GAME01")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("abcdefghijklmnop")
	botPlayer, _ := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	g, _ := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	// Make host the announcer by advancing if needed
	announcer := g.CurrentAnnouncer()
	if announcer == botPlayer.ID {
		// Bot announces first, then both need to place
		s.mockRandom.QueueIntn(0) // letter 'A'
		s.mockRandom.QueueIntn(0) // bot picks position index 0 = (0,0)
		actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
		s.Require().NoError(err)
		// Should have announced + placed
		s.GreaterOrEqual(len(actions), 2)
		s.Equal(bot.ActionAnnounce, actions[0].Type)
		s.Equal(bot.ActionPlace, actions[1].Type)
	} else {
		// Host announces, then bot should place
		_ = s.gameController.AnnounceLetter(s.ctx, g.ID, host.ID, 'A')

		s.mockRandom.QueueIntn(0) // bot picks position index 0
		actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
		s.Require().NoError(err)
		s.Require().NotEmpty(actions)
		s.Equal(bot.ActionPlace, actions[0].Type)
		s.Equal(botPlayer.ID, actions[0].PlayerID)
	}
}

func (s *ServiceSuite) TestProcessBotActions_HumanAnnouncer() {
	s.mockRandom.QueueString("LOBBY1", "GAME01")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("abcdefghijklmnop")
	_, _ = s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	g, _ := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	if g.CurrentAnnouncer() == host.ID {
		// Human is announcer, bot should do nothing
		actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
		s.Require().NoError(err)
		s.Empty(actions)
	}
}

func (s *ServiceSuite) TestProcessBotActions_GameAlreadyScoring() {
	s.mockRandom.QueueString("LOBBY1", "GAME01")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)
	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	g, _ := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	// Complete the game manually
	for turn := 0; turn < 4; turn++ {
		_ = s.gameController.AnnounceLetter(s.ctx, g.ID, host.ID, rune('A'+turn))
		_ = s.gameController.PlaceLetter(s.ctx, g.ID, host.ID, model.Position{Row: turn / 2, Col: turn % 2})
	}

	actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
	s.Require().NoError(err)
	s.Empty(actions)
}

func (s *ServiceSuite) TestProcessBotActions_CascadingTurns() {
	// 1 human host + 2 bots, 2x2 grid
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	s.mockRandom.QueueString("bot1abcdefghijkl")
	bot1, _ := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	s.mockRandom.QueueString("bot2abcdefghijkl")
	bot2, _ := s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.mockRandom.QueueString("GAME01")
	g, err := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)
	s.Require().NoError(err)

	// Host announces first turn
	_ = s.gameController.AnnounceLetter(s.ctx, g.ID, host.ID, 'A')
	// Host places
	_ = s.gameController.PlaceLetter(s.ctx, g.ID, host.ID, model.Position{Row: 0, Col: 0})

	// Queue random values for both bots to place, and for subsequent bot turns
	// Bots need to place for turn 1
	s.mockRandom.QueueIntn(0) // bot1 position (first empty)
	s.mockRandom.QueueIntn(0) // bot2 position (first empty)

	// After all place turn 1, announcer rotates. If next announcer is a bot,
	// they'll announce and then all bots place again, etc.
	// Queue enough random values for the cascade
	for range 20 {
		s.mockRandom.QueueIntn(0) // letters and positions
	}

	actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
	s.Require().NoError(err)

	// Should have multiple actions from the cascade
	s.NotEmpty(actions)

	// Verify game advanced
	updatedGame, _ := s.gameController.GetGame(s.ctx, g.ID)
	// At minimum, turn should have advanced past turn 0
	s.Greater(updatedGame.CurrentTurn, 0)

	// Verify bot actions have correct player IDs
	for _, action := range actions {
		if action.Type == bot.ActionAnnounce || action.Type == bot.ActionPlace {
			s.True(action.PlayerID == bot1.ID || action.PlayerID == bot2.ID,
				"action player should be a bot")
		}
	}
}

func (s *ServiceSuite) TestProcessBotActions_MixedPlayers() {
	// 2 humans + 1 bot on 2x2 grid
	s.mockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lob, _ := s.lobbyController.CreateLobby(s.ctx, host)

	human2 := s.createPlayer("human2", "Human 2")
	_ = s.lobbyController.JoinLobby(s.ctx, lob.Code, human2)

	s.mockRandom.QueueString("bot1abcdefghijkl")
	_, _ = s.botService.AddBotToLobby(s.ctx, lob.Code, host.ID)

	_ = s.lobbyController.UpdateConfig(s.ctx, lob.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.mockRandom.QueueString("GAME01")
	g, _ := s.lobbyController.StartGame(s.ctx, lob.Code, host.ID)

	// Host announces
	_ = s.gameController.AnnounceLetter(s.ctx, g.ID, host.ID, 'X')

	// Queue random for bot placement
	s.mockRandom.QueueIntn(0)

	actions, err := s.botService.ProcessBotActions(s.ctx, g.ID)
	s.Require().NoError(err)

	// Bot places, but human2 still needs to place, so loop stops
	if len(actions) > 0 {
		s.Equal(bot.ActionPlace, actions[0].Type)
	}

	// Game should still be in placing state (human2 hasn't placed)
	updatedGame, _ := s.gameController.GetGame(s.ctx, g.ID)
	s.Equal(model.GameStatePlacing, updatedGame.State)
}
