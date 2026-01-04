package mocks

import (
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
)

// MockClock is a mock implementation of Clock for testing
type MockClock struct {
	CurrentTime time.Time
}

// Ensure MockClock implements Clock
var _ clock.Clock = (*MockClock)(nil)

// NewMockClock creates a MockClock set to the given time
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{CurrentTime: t}
}

// Now returns the mocked current time
func (c *MockClock) Now() time.Time {
	return c.CurrentTime
}

// Advance moves the clock forward by the given duration
func (c *MockClock) Advance(d time.Duration) {
	c.CurrentTime = c.CurrentTime.Add(d)
}

// Set sets the clock to the given time
func (c *MockClock) Set(t time.Time) {
	c.CurrentTime = t
}
