package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid or expired session")
	ErrUsernameExists     = errors.New("username already exists")
)

// Session represents an authenticated session
type Session struct {
	Token     string
	PlayerID  model.PlayerID
	Player    model.Player
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Service handles authentication and session management
type Service struct {
	storage storage.Storage
	clock   clock.Clock

	mu       sync.RWMutex
	sessions map[string]*Session

	sessionDuration time.Duration
}

// Config holds configuration for the auth service
type Config struct {
	SessionDuration time.Duration
}

// DefaultConfig returns default auth configuration
func DefaultConfig() Config {
	return Config{
		SessionDuration: 24 * time.Hour,
	}
}

// New creates a new AuthService
func New(storage storage.Storage, clock clock.Clock, cfg Config) *Service {
	if cfg.SessionDuration == 0 {
		cfg.SessionDuration = DefaultConfig().SessionDuration
	}
	return &Service{
		storage:         storage,
		clock:           clock,
		sessions:        make(map[string]*Session),
		sessionDuration: cfg.SessionDuration,
	}
}

// CreateGuestPlayer creates an anonymous player and session
func (s *Service) CreateGuestPlayer(ctx context.Context, displayName string) (*Session, error) {
	playerID := model.PlayerID(s.generateID("p_"))
	now := s.clock.Now()

	player := &model.Player{
		ID:          playerID,
		DisplayName: displayName,
		IsGuest:     true,
		CreatedAt:   now,
	}

	if err := s.storage.SavePlayer(ctx, player); err != nil {
		return nil, err
	}

	return s.createSession(player)
}

// RegisterPlayer creates a registered player account and session
func (s *Service) RegisterPlayer(ctx context.Context, username, password, displayName string) (*Session, error) {
	// Check if username exists
	_, err := s.storage.GetRegisteredPlayerByUsername(ctx, username)
	if err == nil {
		return nil, ErrUsernameExists
	}
	if !errors.Is(err, model.ErrPlayerNotFound) {
		return nil, err
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	playerID := model.PlayerID(s.generateID("p_"))
	now := s.clock.Now()

	player := &model.Player{
		ID:          playerID,
		DisplayName: displayName,
		IsGuest:     false,
		CreatedAt:   now,
	}

	registeredPlayer := &model.RegisteredPlayer{
		PlayerID:     playerID,
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.storage.SavePlayer(ctx, player); err != nil {
		return nil, err
	}

	if err := s.storage.SaveRegisteredPlayer(ctx, registeredPlayer); err != nil {
		return nil, err
	}

	return s.createSession(player)
}

// Login authenticates a registered player and creates a session
func (s *Service) Login(ctx context.Context, username, password string) (*Session, error) {
	rp, err := s.storage.GetRegisteredPlayerByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrPlayerNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rp.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	player, err := s.storage.GetPlayer(ctx, rp.PlayerID)
	if err != nil {
		return nil, err
	}

	return s.createSession(player)
}

// ValidateSession checks if a session token is valid and returns the session
func (s *Service) ValidateSession(token string) (*Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrInvalidSession
	}

	if s.clock.Now().After(session.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, token)
		s.mu.Unlock()
		return nil, ErrInvalidSession
	}

	return session, nil
}

// InvalidateSession removes a session
func (s *Service) InvalidateSession(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// GetPlayer returns the player for a session token
func (s *Service) GetPlayer(token string) (*model.Player, error) {
	session, err := s.ValidateSession(token)
	if err != nil {
		return nil, err
	}
	return &session.Player, nil
}

// createSession creates a new session for a player
func (s *Service) createSession(player *model.Player) (*Session, error) {
	token := s.generateID("sess_")
	now := s.clock.Now()

	session := &Session{
		Token:     token,
		PlayerID:  player.ID,
		Player:    *player,
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionDuration),
	}

	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()

	return session, nil
}

// generateID generates a random ID with a prefix
func (s *Service) generateID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

// CleanExpiredSessions removes expired sessions (call periodically)
func (s *Service) CleanExpiredSessions() {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
}
