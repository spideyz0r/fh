package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	client openai.Client
	model  openai.ChatModel
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(modelName string) (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// Map model name to openai.ChatModel constant
	var model openai.ChatModel
	switch modelName {
	case "gpt-4o":
		model = openai.ChatModelGPT4o
	case "gpt-4o-mini":
		model = openai.ChatModelGPT4oMini
	case "gpt-4":
		model = openai.ChatModelGPT4
	case "gpt-3.5-turbo":
		model = openai.ChatModelGPT3_5Turbo
	default:
		// Default to gpt-4o-mini or treat as custom model
		model = openai.ChatModel(modelName)
	}

	return &OpenAIClient{
		client: client,
		model:  model,
	}, nil
}

// Query sends a prompt to OpenAI and returns the response
func (c *OpenAIClient) Query(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
