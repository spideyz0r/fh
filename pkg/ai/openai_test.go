package ai

import (
	"os"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
)

func TestNewOpenAIClient_MissingAPIKey(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Unset the API key
	os.Unsetenv("OPENAI_API_KEY")

	client, err := NewOpenAIClient("gpt-4o-mini")

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY")
}

func TestNewOpenAIClient_WithAPIKey(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Set a test API key
	os.Setenv("OPENAI_API_KEY", "sk-test-key-12345")

	client, err := NewOpenAIClient("gpt-4o-mini")

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, openai.ChatModelGPT4oMini, client.model)
}

func TestNewOpenAIClient_ModelMapping(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Set a test API key
	os.Setenv("OPENAI_API_KEY", "sk-test-key-12345")

	tests := []struct {
		name          string
		modelName     string
		expectedModel openai.ChatModel
	}{
		{
			name:          "gpt-4o",
			modelName:     "gpt-4o",
			expectedModel: openai.ChatModelGPT4o,
		},
		{
			name:          "gpt-4o-mini",
			modelName:     "gpt-4o-mini",
			expectedModel: openai.ChatModelGPT4oMini,
		},
		{
			name:          "gpt-4",
			modelName:     "gpt-4",
			expectedModel: openai.ChatModelGPT4,
		},
		{
			name:          "gpt-3.5-turbo",
			modelName:     "gpt-3.5-turbo",
			expectedModel: openai.ChatModelGPT3_5Turbo,
		},
		{
			name:          "custom model",
			modelName:     "custom-model-name",
			expectedModel: openai.ChatModel("custom-model-name"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.modelName)
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.expectedModel, client.model)
		})
	}
}

func TestNewOpenAIClient_EmptyModelName(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Set a test API key
	os.Setenv("OPENAI_API_KEY", "sk-test-key-12345")

	// Empty model name should be treated as custom model
	client, err := NewOpenAIClient("")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, openai.ChatModel(""), client.model)
}

func TestOpenAIClient_Structure(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	// Set a test API key
	os.Setenv("OPENAI_API_KEY", "sk-test-key-12345")

	client, err := NewOpenAIClient("gpt-4o-mini")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Verify client structure
	assert.NotNil(t, client.client)
	assert.NotEmpty(t, client.model)
}

// Note: Testing Query() would require mocking the OpenAI API
// which is complex. Integration tests should cover this.
// For unit tests, we've verified:
// 1. API key validation
// 2. Model mapping
// 3. Client initialization
