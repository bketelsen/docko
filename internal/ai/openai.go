package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// aiResponseSchema defines the JSON schema for OpenAI structured output
// Manually defined to ensure compatibility with OpenAI's strict schema requirements
var aiResponseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"tags": map[string]any{
			"type":        "array",
			"description": "Array of suggested tags for the document",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":       map[string]any{"type": "string", "description": "The tag name"},
					"confidence": map[string]any{"type": "number", "description": "Confidence score from 0.0 to 1.0"},
					"reasoning":  map[string]any{"type": "string", "description": "Brief explanation for this suggestion"},
				},
				"required":             []string{"name", "confidence", "reasoning"},
				"additionalProperties": false,
			},
		},
		"correspondent": map[string]any{
			"type":        []string{"object", "null"},
			"description": "Suggested correspondent (sender/organization), or null if unclear",
			"properties": map[string]any{
				"name":       map[string]any{"type": "string", "description": "The correspondent name"},
				"confidence": map[string]any{"type": "number", "description": "Confidence score from 0.0 to 1.0"},
				"reasoning":  map[string]any{"type": "string", "description": "Brief explanation for this suggestion"},
			},
			"required":             []string{"name", "confidence", "reasoning"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"tags", "correspondent"},
	"additionalProperties": false,
}

// OpenAIProvider implements Provider using the OpenAI API
type OpenAIProvider struct {
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &OpenAIProvider{} // Not configured
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{
		client: &client,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Available() bool {
	return p.client != nil
}

func (p *OpenAIProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
	if !p.Available() {
		return nil, fmt.Errorf("openai provider not configured")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	prompt := BuildPrompt(req)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "document_analysis",
		Description: openai.String("AI analysis of document with tag and correspondent suggestions"),
		Schema:      aiResponseSchema,
		Strict:      openai.Bool(true),
	}

	chatResp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(SystemPrompt),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		MaxTokens: openai.Int(int64(maxTokens)),
	})
	if err != nil {
		return nil, fmt.Errorf("openai completion: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &aiResp); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	return &AnalyzeResponse{
		Suggestions: ConvertToSuggestions(aiResp, req.ExistingTags, req.Correspondents),
		Usage: Usage{
			InputTokens:  int(chatResp.Usage.PromptTokens),
			OutputTokens: int(chatResp.Usage.CompletionTokens),
			Model:        chatResp.Model,
		},
	}, nil
}
