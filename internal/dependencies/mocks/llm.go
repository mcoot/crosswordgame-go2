package mocks

import (
	"context"
	"fmt"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/llm"
)

// MockLLMClient is a mock implementation of llm.Client for testing
type MockLLMClient struct {
	// Responses is a queue of responses to return
	Responses []*llm.Response
	// Errors is a queue of errors to return
	Errors []error
	// Prompts records all prompts received
	Prompts []string

	callIndex int
}

// Ensure MockLLMClient implements llm.Client
var _ llm.Client = (*MockLLMClient)(nil)

// NewMockLLMClient creates a new MockLLMClient
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{}
}

// GenerateContent returns the next queued response or error
func (m *MockLLMClient) GenerateContent(_ context.Context, prompt string) (*llm.Response, error) {
	m.Prompts = append(m.Prompts, prompt)

	if m.callIndex >= len(m.Responses) && m.callIndex >= len(m.Errors) {
		return nil, fmt.Errorf("no more queued responses")
	}

	var err error
	if m.callIndex < len(m.Errors) {
		err = m.Errors[m.callIndex]
	}

	var resp *llm.Response
	if m.callIndex < len(m.Responses) {
		resp = m.Responses[m.callIndex]
	}

	m.callIndex++

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// QueueResponse adds a successful response to the queue
func (m *MockLLMClient) QueueResponse(text string) {
	m.Responses = append(m.Responses, &llm.Response{Text: text})
	// Keep errors slice same length with nil entries
	for len(m.Errors) < len(m.Responses) {
		m.Errors = append(m.Errors, nil)
	}
}

// QueueError adds an error response to the queue
func (m *MockLLMClient) QueueError(err error) {
	m.Errors = append(m.Errors, err)
	// Keep responses slice same length with nil entries
	for len(m.Responses) < len(m.Errors) {
		m.Responses = append(m.Responses, nil)
	}
}

// Reset clears all queued responses and recorded prompts
func (m *MockLLMClient) Reset() {
	m.Responses = nil
	m.Errors = nil
	m.Prompts = nil
	m.callIndex = 0
}
