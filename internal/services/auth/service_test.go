package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
)

type ServiceSuite struct {
	suite.Suite
	storage *memory.Storage
	clock   *mocks.MockClock
	service *Service
	ctx     context.Context
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.storage = memory.New()
	s.clock = mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	s.service = New(s.storage, s.clock, DefaultConfig())
	s.ctx = context.Background()
}

// CreateGuestPlayer tests

func (s *ServiceSuite) TestCreateGuestPlayerSucceeds() {
	session, err := s.service.CreateGuestPlayer(s.ctx, "Alice")
	s.Require().NoError(err)

	s.NotEmpty(session.Token)
	s.Equal("Alice", session.Player.DisplayName)
	s.True(session.Player.IsGuest)
	s.NotEmpty(session.PlayerID)
}

func (s *ServiceSuite) TestCreateGuestPlayerPersistsPlayer() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	player, err := s.storage.GetPlayer(s.ctx, session.PlayerID)
	s.Require().NoError(err)
	s.Equal("Alice", player.DisplayName)
}

func (s *ServiceSuite) TestCreateGuestPlayerSessionIsValid() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	validated, err := s.service.ValidateSession(session.Token)
	s.Require().NoError(err)
	s.Equal(session.PlayerID, validated.PlayerID)
}

// RegisterPlayer tests

func (s *ServiceSuite) TestRegisterPlayerSucceeds() {
	session, err := s.service.RegisterPlayer(s.ctx, "alice", "password123", "Alice")
	s.Require().NoError(err)

	s.NotEmpty(session.Token)
	s.Equal("Alice", session.Player.DisplayName)
	s.False(session.Player.IsGuest)
}

func (s *ServiceSuite) TestRegisterPlayerPersistsRegistration() {
	_, _ = s.service.RegisterPlayer(s.ctx, "alice", "password123", "Alice")

	rp, err := s.storage.GetRegisteredPlayerByUsername(s.ctx, "alice")
	s.Require().NoError(err)
	s.Equal("alice", rp.Username)
	s.NotEmpty(rp.PasswordHash)
	s.NotEqual("password123", rp.PasswordHash) // Should be hashed
}

func (s *ServiceSuite) TestRegisterPlayerFailsIfUsernameExists() {
	_, _ = s.service.RegisterPlayer(s.ctx, "alice", "password123", "Alice")

	_, err := s.service.RegisterPlayer(s.ctx, "alice", "different", "Alice2")
	s.ErrorIs(err, ErrUsernameExists)
}

// Login tests

func (s *ServiceSuite) TestLoginSucceeds() {
	_, _ = s.service.RegisterPlayer(s.ctx, "alice", "password123", "Alice")

	session, err := s.service.Login(s.ctx, "alice", "password123")
	s.Require().NoError(err)

	s.NotEmpty(session.Token)
	s.Equal("Alice", session.Player.DisplayName)
}

func (s *ServiceSuite) TestLoginFailsWithWrongPassword() {
	_, _ = s.service.RegisterPlayer(s.ctx, "alice", "password123", "Alice")

	_, err := s.service.Login(s.ctx, "alice", "wrongpassword")
	s.ErrorIs(err, ErrInvalidCredentials)
}

func (s *ServiceSuite) TestLoginFailsWithUnknownUser() {
	_, err := s.service.Login(s.ctx, "nobody", "password123")
	s.ErrorIs(err, ErrInvalidCredentials)
}

// ValidateSession tests

func (s *ServiceSuite) TestValidateSessionSucceeds() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	validated, err := s.service.ValidateSession(session.Token)
	s.Require().NoError(err)
	s.Equal(session.Token, validated.Token)
}

func (s *ServiceSuite) TestValidateSessionFailsWithInvalidToken() {
	_, err := s.service.ValidateSession("invalid_token")
	s.ErrorIs(err, ErrInvalidSession)
}

func (s *ServiceSuite) TestValidateSessionFailsWhenExpired() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	// Advance time past expiration
	s.clock.Advance(25 * time.Hour)

	_, err := s.service.ValidateSession(session.Token)
	s.ErrorIs(err, ErrInvalidSession)
}

// InvalidateSession tests

func (s *ServiceSuite) TestInvalidateSessionRemovesSession() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	s.service.InvalidateSession(session.Token)

	_, err := s.service.ValidateSession(session.Token)
	s.ErrorIs(err, ErrInvalidSession)
}

func (s *ServiceSuite) TestInvalidateSessionNoopForUnknownToken() {
	// Should not panic
	s.service.InvalidateSession("unknown_token")
}

// GetPlayer tests

func (s *ServiceSuite) TestGetPlayerSucceeds() {
	session, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	player, err := s.service.GetPlayer(session.Token)
	s.Require().NoError(err)
	s.Equal("Alice", player.DisplayName)
}

func (s *ServiceSuite) TestGetPlayerFailsWithInvalidToken() {
	_, err := s.service.GetPlayer("invalid_token")
	s.ErrorIs(err, ErrInvalidSession)
}

// CleanExpiredSessions tests

func (s *ServiceSuite) TestCleanExpiredSessionsRemovesExpired() {
	session1, _ := s.service.CreateGuestPlayer(s.ctx, "Alice")

	// Advance time so session1 expires
	s.clock.Advance(25 * time.Hour)

	// Create a new session (not expired)
	session2, _ := s.service.CreateGuestPlayer(s.ctx, "Bob")

	s.service.CleanExpiredSessions()

	// session1 should be gone
	_, err := s.service.ValidateSession(session1.Token)
	s.ErrorIs(err, ErrInvalidSession)

	// session2 should still be valid
	_, err = s.service.ValidateSession(session2.Token)
	s.NoError(err)
}
