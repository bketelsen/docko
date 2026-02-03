package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements Provider using the Anthropic Claude API
type AnthropicProvider struct {
	client *anthropic.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return &AnthropicProvider{} // Not configured
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		client: &client,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Available() bool {
	return p.client != nil
}

func (p *AnthropicProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
	if !p.Available() {
		return nil, fmt.Errorf("anthropic provider not configured")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	prompt := BuildPrompt(req)

	// Build full prompt with JSON instruction since Claude doesn't have structured output mode
	fullPrompt := fmt.Sprintf("%s\n\nRespond with valid JSON only, no markdown code blocks.\n\n%s", SystemPrompt, prompt)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-haiku-4-5-20251101",
		MaxTokens: int64(maxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fullPrompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic message: %w", err)
	}

	// Extract text content from response
	var content string
	for _, block := range message.Content {
		if block.Type == "text" {
			content = block.Text
			break
		}
	}

	if content == "" {
		return nil, fmt.Errorf("anthropic returned no text content")
	}

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(content), &aiResp); err != nil {
		return nil, fmt.Errorf("parse anthropic response: %w (raw: %s)", err, content)
	}

	return &AnalyzeResponse{
		Suggestions: ConvertToSuggestions(aiResp, req.ExistingTags, req.Correspondents),
		Usage: Usage{
			InputTokens:  int(message.Usage.InputTokens),
			OutputTokens: int(message.Usage.OutputTokens),
			Model:        string(message.Model),
		},
	}, nil
}
