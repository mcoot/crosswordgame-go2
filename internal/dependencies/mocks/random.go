package mocks

import (
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
)

// MockRandom is a mock implementation of Random for testing
type MockRandom struct {
	// IntnResults is a queue of results to return from Intn
	IntnResults []int
	intnIndex   int

	// StringResults is a queue of results to return from String
	StringResults []string
	stringIndex   int
}

// Ensure MockRandom implements Random
var _ random.Random = (*MockRandom)(nil)

// NewMockRandom creates a new MockRandom
func NewMockRandom() *MockRandom {
	return &MockRandom{}
}

// Intn returns the next queued result, or 0 if none remaining
func (r *MockRandom) Intn(n int) int {
	if r.intnIndex >= len(r.IntnResults) {
		return 0
	}
	result := r.IntnResults[r.intnIndex]
	r.intnIndex++
	return result
}

// String returns the next queued result, or empty string if none remaining
func (r *MockRandom) String(length int, alphabet string) string {
	if r.stringIndex >= len(r.StringResults) {
		return ""
	}
	result := r.StringResults[r.stringIndex]
	r.stringIndex++
	return result
}

// QueueIntn adds values to the Intn result queue
func (r *MockRandom) QueueIntn(values ...int) {
	r.IntnResults = append(r.IntnResults, values...)
}

// QueueString adds values to the String result queue
func (r *MockRandom) QueueString(values ...string) {
	r.StringResults = append(r.StringResults, values...)
}

// Reset clears all queued results
func (r *MockRandom) Reset() {
	r.IntnResults = nil
	r.intnIndex = 0
	r.StringResults = nil
	r.stringIndex = 0
}
