package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

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

	chatResp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(SystemPrompt + "\n\nRespond with valid JSON only."),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
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
