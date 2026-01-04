package clock

import "time"

// Clock provides time operations that can be mocked for testing
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the system clock
type RealClock struct{}

// New creates a new RealClock
func New() *RealClock {
	return &RealClock{}
}

// Now returns the current time
func (c *RealClock) Now() time.Time {
	return time.Now()
}
