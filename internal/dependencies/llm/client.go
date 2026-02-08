package llm

import "context"

// Response holds the text output from an LLM call
type Response struct {
	Text string
}

// Client provides LLM text generation that can be mocked for testing
type Client interface {
	// GenerateContent sends a prompt and returns the generated text
	GenerateContent(ctx context.Context, prompt string) (*Response, error)
}
