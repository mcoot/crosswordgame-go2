package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiClient wraps the Google GenAI SDK to implement Client
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient creates a GeminiClient with the given API key and model name
func NewGeminiClient(ctx context.Context, apiKey string, model string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("creating genai client: %w", err)
	}
	return &GeminiClient{client: client, model: model}, nil
}

// GenerateContent sends a prompt to the Gemini model and returns the text response
func (g *GeminiClient) GenerateContent(ctx context.Context, prompt string) (*Response, error) {
	result, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("empty response from model")
	}

	return &Response{Text: text}, nil
}
