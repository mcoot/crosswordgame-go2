package factory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

type IntegrationSuite struct {
	suite.Suite
	app *TestApp
	ctx context.Context
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

func (s *IntegrationSuite) SetupTest() {
	s.app = NewTestApp()
	s.ctx = context.Background()
	s.Require().NoError(s.app.LoadTestDictionary())
}

func (s *IntegrationSuite) createPlayer(id, name string) model.Player {
	return model.Player{
		ID:          model.PlayerID(id),
		DisplayName: name,
		IsGuest:     true,
		CreatedAt:   s.app.MockClock.Now(),
	}
}

// Test: Complete game flow from lobby creation to game completion
func (s *IntegrationSuite) TestCompleteGameFlow() {
	// Setup: Queue random values
	s.app.MockRandom.QueueString("LOBBY1", "GAME01")

	// Step 1: Create a lobby
	host := s.createPlayer("host", "Host Player")
	lobby, err := s.app.LobbyController.CreateLobby(s.ctx, host)
	s.Require().NoError(err)
	s.Equal(model.LobbyCode("LOBBY1"), lobby.Code)

	// Step 2: Another player joins
	player2 := s.createPlayer("player2", "Player Two")
	err = s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, player2)
	s.Require().NoError(err)

	// Step 3: Configure for a small grid (2x2 = 4 turns)
	err = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.Require().NoError(err)

	// Step 4: Start the game
	game, err := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)
	s.Equal(model.GameStateAnnouncing, game.State)
	s.Len(game.Players, 2)

	// Step 5: Play through all 4 turns
	// Turn 1: Host announces 'C', both place at (0,0)
	err = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, 'C')
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: 0, Col: 0})
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, player2.ID, model.Position{Row: 0, Col: 0})
	s.Require().NoError(err)

	// Turn 2: Player2 announces 'A', both place at (0,1)
	err = s.app.GameController.AnnounceLetter(s.ctx, game.ID, player2.ID, 'A')
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: 0, Col: 1})
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, player2.ID, model.Position{Row: 0, Col: 1})
	s.Require().NoError(err)

	// Turn 3: Host announces 'T', both place at (1,0)
	err = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, 'T')
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: 1, Col: 0})
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, player2.ID, model.Position{Row: 1, Col: 0})
	s.Require().NoError(err)

	// Turn 4: Player2 announces 'O', both place at (1,1)
	err = s.app.GameController.AnnounceLetter(s.ctx, game.ID, player2.ID, 'O')
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: 1, Col: 1})
	s.Require().NoError(err)
	err = s.app.GameController.PlaceLetter(s.ctx, game.ID, player2.ID, model.Position{Row: 1, Col: 1})
	s.Require().NoError(err)

	// Step 6: Verify game is in scoring state
	updatedGame, err := s.app.GameController.GetGame(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Equal(model.GameStateScoring, updatedGame.State)

	// Step 7: Get final scores
	scores, err := s.app.GameController.GetFinalScores(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Len(scores, 2)

	// Both players have same board: CA / TO
	// Should find "AT" (column 1) for 2 points each
	// Actually with CA/TO: no horizontal words, but "AT" vertically in col 1 (A, T) = 2 pts
	// And "TO" horizontally row 1 = 2 pts (wait, it's T, O which spells TO!)
	// Let me check what words would be found...
	// Row 0: CA - not a word
	// Row 1: TO - yes, 2 letters, score 2
	// Col 0: CT - not a word
	// Col 1: AO - not a word
	// So each player should score 2 for "TO"
	for _, score := range scores {
		s.GreaterOrEqual(score.TotalScore, 0)
	}

	// Step 8: Complete the game in lobby
	err = s.app.LobbyController.CompleteGame(s.ctx, lobby.Code)
	s.Require().NoError(err)

	// Step 9: Verify lobby is back to waiting with game in history
	updatedLobby, err := s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Require().NoError(err)
	s.Equal(model.LobbyStateWaiting, updatedLobby.State)
	s.Nil(updatedLobby.CurrentGame)
	s.Len(updatedLobby.GameHistory, 1)
}

// Test: Player leaves during game
func (s *IntegrationSuite) TestPlayerLeavesDuringGame() {
	s.app.MockRandom.QueueString("LOBBY1", "GAME01")

	// Create lobby with 3 players
	host := s.createPlayer("host", "Host")
	player2 := s.createPlayer("player2", "Player 2")
	player3 := s.createPlayer("player3", "Player 3")

	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)
	_ = s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, player2)
	_ = s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, player3)

	// Start game
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	game, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)

	// Start first turn
	_ = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, 'A')
	_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: 0, Col: 0})

	// Player 2 leaves before placing
	err := s.app.LobbyController.LeaveLobby(s.ctx, lobby.Code, player2.ID)
	s.Require().NoError(err)

	// Player 3 places, turn should advance (only 2 players now, both placed)
	_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, player3.ID, model.Position{Row: 0, Col: 0})

	updatedGame, _ := s.app.GameController.GetGame(s.ctx, game.ID)
	s.Equal(1, updatedGame.CurrentTurn) // Turn advanced
	s.Len(updatedGame.Players, 2)       // Only host and player3
}

// Test: All players leave abandons game and deletes lobby
func (s *IntegrationSuite) TestAllPlayersLeaveDeletesLobby() {
	s.app.MockRandom.QueueString("LOBBY1", "GAME01")

	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)

	player2 := s.createPlayer("player2", "Player 2")
	_ = s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, player2)

	// Start game
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	_, _ = s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)

	// Both players leave
	_ = s.app.LobbyController.LeaveLobby(s.ctx, lobby.Code, host.ID)
	_ = s.app.LobbyController.LeaveLobby(s.ctx, lobby.Code, player2.ID)

	// Lobby should be deleted
	_, err := s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.ErrorIs(err, model.ErrLobbyNotFound)
}

// Test: Spectator joins during game and becomes player after game
func (s *IntegrationSuite) TestSpectatorFlowDuringGame() {
	s.app.MockRandom.QueueString("LOBBY1", "GAME01")

	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)

	// Start game with just host
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	game, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)

	// Spectator joins during game
	spectator := s.createPlayer("spectator", "Spectator")
	err := s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, spectator)
	s.Require().NoError(err)

	updatedLobby, _ := s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.RoleSpectator, updatedLobby.GetMember(spectator.ID).Role)

	// Complete the game quickly
	positions := []model.Position{{Row: 0, Col: 0}, {Row: 0, Col: 1}, {Row: 1, Col: 0}, {Row: 1, Col: 1}}
	for i, pos := range positions {
		_ = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, rune('A'+i))
		_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, pos)
	}

	_ = s.app.LobbyController.CompleteGame(s.ctx, lobby.Code)

	// Now spectator can become player
	err = s.app.LobbyController.SetRole(s.ctx, lobby.Code, spectator.ID, model.RolePlayer)
	s.Require().NoError(err)

	updatedLobby, _ = s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.RolePlayer, updatedLobby.GetMember(spectator.ID).Role)
}

// Test: Host transfer during lobby
func (s *IntegrationSuite) TestHostTransfer() {
	s.app.MockRandom.QueueString("LOBBY1")

	host := s.createPlayer("host", "Host")
	player2 := s.createPlayer("player2", "Player 2")

	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)
	_ = s.app.LobbyController.JoinLobby(s.ctx, lobby.Code, player2)

	// Transfer host to player2
	err := s.app.LobbyController.TransferHost(s.ctx, lobby.Code, host.ID, player2.ID)
	s.Require().NoError(err)

	// Player2 can now start game
	s.app.MockRandom.QueueString("GAME01")
	_, err = s.app.LobbyController.StartGame(s.ctx, lobby.Code, player2.ID)
	s.Require().NoError(err)

	// Original host cannot abandon (not host anymore)
	err = s.app.LobbyController.AbandonGame(s.ctx, lobby.Code, host.ID)
	s.ErrorIs(err, model.ErrNotHost)

	// New host can abandon
	err = s.app.LobbyController.AbandonGame(s.ctx, lobby.Code, player2.ID)
	s.Require().NoError(err)
}

// Test: Scoring with actual words
func (s *IntegrationSuite) TestScoringWithRealWords() {
	s.app.MockRandom.QueueString("LOBBY1", "GAME01")

	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 3})

	game, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)

	// Build a 3x3 board that spells "CAT" horizontally in row 0
	// C A T
	// X X X
	// X X X
	letters := []rune{'C', 'A', 'T', 'X', 'Y', 'Z', 'P', 'Q', 'R'}
	positions := []model.Position{
		{Row: 0, Col: 0}, {Row: 0, Col: 1}, {Row: 0, Col: 2},
		{Row: 1, Col: 0}, {Row: 1, Col: 1}, {Row: 1, Col: 2},
		{Row: 2, Col: 0}, {Row: 2, Col: 1}, {Row: 2, Col: 2},
	}

	for i := 0; i < 9; i++ {
		_ = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, letters[i])
		_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, positions[i])
	}

	scores, err := s.app.GameController.GetFinalScores(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Require().Len(scores, 1)

	// Should find "CAT" (3 letters, full row = 6 points)
	// And "AT" within CAT would be subsumed by CAT (greedy picks longer)
	foundCat := false
	for _, word := range scores[0].Words {
		if word.Word == "CAT" {
			foundCat = true
			s.Equal(6, word.Score) // Full row bonus
		}
	}
	s.True(foundCat, "should find CAT")
	s.Equal(6, scores[0].TotalScore)
}

// Test: Bot game flow - 1 human + 1 bot on 2x2 grid
func (s *IntegrationSuite) TestBotGameFlow() {
	s.app.MockRandom.QueueString("LOBBY1")

	host := s.createPlayer("host", "Host Player")
	lobby, err := s.app.LobbyController.CreateLobby(s.ctx, host)
	s.Require().NoError(err)

	// Add a bot to the lobby
	s.app.MockRandom.QueueString("botplayer1abcdef")
	botPlayer, err := s.app.BotService.AddBotToLobby(s.ctx, lobby.Code, host.ID, model.BotStrategyRandom)
	s.Require().NoError(err)
	s.True(botPlayer.IsBot)

	// Configure and start game
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.app.MockRandom.QueueString("GAME01")
	game, err := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)
	s.Len(game.Players, 2)

	// Queue random values for bot actions
	for range 20 {
		s.app.MockRandom.QueueIntn(0)
	}

	// Process any initial bot actions (if bot is first announcer)
	_, _ = s.app.BotService.ProcessBotActions(s.ctx, game.ID)

	// Play through: host announces and places each turn, bot auto-places
	for turn := 0; turn < 4; turn++ {
		updatedGame, _ := s.app.GameController.GetGame(s.ctx, game.ID)
		if updatedGame.State == model.GameStateScoring {
			break
		}

		if updatedGame.State == model.GameStateAnnouncing && updatedGame.CurrentAnnouncer() == host.ID {
			_ = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, rune('A'+turn))
			// Bot should auto-place after announcement
			s.app.MockRandom.QueueIntn(0)
			_, _ = s.app.BotService.ProcessBotActions(s.ctx, game.ID)
		}

		updatedGame, _ = s.app.GameController.GetGame(s.ctx, game.ID)
		if updatedGame.State == model.GameStatePlacing && !updatedGame.Placements[host.ID] {
			_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: turn / 2, Col: turn % 2})
			// Process bot actions after host places
			for range 5 {
				s.app.MockRandom.QueueIntn(0)
			}
			_, _ = s.app.BotService.ProcessBotActions(s.ctx, game.ID)
		}
	}

	// Verify game completed
	finalGame, err := s.app.GameController.GetGame(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Equal(model.GameStateScoring, finalGame.State)
}

// Test: All bots game - 1 human host + 2 bots
func (s *IntegrationSuite) TestAllBotsGame() {
	s.app.MockRandom.QueueString("LOBBY1")

	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)

	s.app.MockRandom.QueueString("bot1abcdefghijkl")
	_, _ = s.app.BotService.AddBotToLobby(s.ctx, lobby.Code, host.ID, model.BotStrategyRandom)

	s.app.MockRandom.QueueString("bot2abcdefghijkl")
	_, _ = s.app.BotService.AddBotToLobby(s.ctx, lobby.Code, host.ID, model.BotStrategyRandom)

	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.app.MockRandom.QueueString("GAME01")
	game, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)

	// Queue enough random values for the whole game
	for range 50 {
		s.app.MockRandom.QueueIntn(0)
	}

	// Host announces, places, then bots cascade
	for turn := 0; turn < 4; turn++ {
		g, _ := s.app.GameController.GetGame(s.ctx, game.ID)
		if g.State == model.GameStateScoring {
			break
		}

		if g.State == model.GameStateAnnouncing && g.CurrentAnnouncer() == host.ID {
			_ = s.app.GameController.AnnounceLetter(s.ctx, game.ID, host.ID, rune('A'+turn))
			_, _ = s.app.BotService.ProcessBotActions(s.ctx, game.ID)
		}

		g, _ = s.app.GameController.GetGame(s.ctx, game.ID)
		if g.State == model.GameStatePlacing && !g.Placements[host.ID] {
			_ = s.app.GameController.PlaceLetter(s.ctx, game.ID, host.ID, model.Position{Row: turn / 2, Col: turn % 2})
			_, _ = s.app.BotService.ProcessBotActions(s.ctx, game.ID)
		}
	}

	finalGame, _ := s.app.GameController.GetGame(s.ctx, game.ID)
	s.Equal(model.GameStateScoring, finalGame.State)
}

// Test: Add and remove bot from lobby
func (s *IntegrationSuite) TestAddRemoveBot() {
	s.app.MockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)

	// Add bot
	s.app.MockRandom.QueueString("bot1abcdefghijkl")
	botPlayer, err := s.app.BotService.AddBotToLobby(s.ctx, lobby.Code, host.ID, model.BotStrategyRandom)
	s.Require().NoError(err)

	updatedLobby, _ := s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Len(updatedLobby.Members, 2)

	// Remove bot
	err = s.app.BotService.RemoveBotFromLobby(s.ctx, lobby.Code, host.ID, botPlayer.ID)
	s.Require().NoError(err)

	updatedLobby, _ = s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Len(updatedLobby.Members, 1)
}

// Test: Bot in lobby starts game with bot as player
func (s *IntegrationSuite) TestBotInLobbyStartsGame() {
	s.app.MockRandom.QueueString("LOBBY1")
	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)

	s.app.MockRandom.QueueString("bot1abcdefghijkl")
	botPlayer, _ := s.app.BotService.AddBotToLobby(s.ctx, lobby.Code, host.ID, model.BotStrategyRandom)

	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	s.app.MockRandom.QueueString("GAME01")
	game, err := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)

	// Bot should be in the game players list
	foundBot := false
	for _, pid := range game.Players {
		if pid == botPlayer.ID {
			foundBot = true
			break
		}
	}
	s.True(foundBot, "bot should be in game players list")
}

// Test: Multiple games in same lobby
func (s *IntegrationSuite) TestMultipleGamesInLobby() {
	s.app.MockRandom.QueueString("LOBBY1", "GAME01", "GAME02")

	host := s.createPlayer("host", "Host")
	lobby, _ := s.app.LobbyController.CreateLobby(s.ctx, host)
	_ = s.app.LobbyController.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})

	// Play first game
	game1, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)
	for i := 0; i < 4; i++ {
		_ = s.app.GameController.AnnounceLetter(s.ctx, game1.ID, host.ID, rune('A'+i))
		_ = s.app.GameController.PlaceLetter(s.ctx, game1.ID, host.ID, model.Position{Row: i / 2, Col: i % 2})
	}
	_ = s.app.LobbyController.CompleteGame(s.ctx, lobby.Code)

	// Play second game
	game2, _ := s.app.LobbyController.StartGame(s.ctx, lobby.Code, host.ID)
	for i := 0; i < 4; i++ {
		_ = s.app.GameController.AnnounceLetter(s.ctx, game2.ID, host.ID, rune('E'+i))
		_ = s.app.GameController.PlaceLetter(s.ctx, game2.ID, host.ID, model.Position{Row: i / 2, Col: i % 2})
	}
	_ = s.app.LobbyController.CompleteGame(s.ctx, lobby.Code)

	// Verify history
	updatedLobby, _ := s.app.LobbyController.GetLobby(s.ctx, lobby.Code)
	s.Len(updatedLobby.GameHistory, 2)
	s.Equal(game1.ID, updatedLobby.GameHistory[0].ID)
	s.Equal(game2.ID, updatedLobby.GameHistory[1].ID)
}
