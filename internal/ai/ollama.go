package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/api"
)

// OllamaProvider implements Provider using a local Ollama instance
type OllamaProvider struct {
	model string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider() *OllamaProvider {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2" // Default model
	}
	return &OllamaProvider{model: model}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Available() bool {
	// Ollama is available if OLLAMA_HOST is set or we assume localhost
	// Connection errors are handled at runtime
	return os.Getenv("OLLAMA_HOST") != "" || true
}

func (p *OllamaProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("create ollama client: %w", err)
	}

	prompt := BuildPrompt(req)

	// Build full prompt with system instruction for Ollama
	fullPrompt := fmt.Sprintf("%s\n\n%s\n\nRespond with valid JSON only.", SystemPrompt, prompt)

	var response strings.Builder
	var evalCount, promptEvalCount int

	stream := false
	err = client.Generate(ctx, &api.GenerateRequest{
		Model:  p.model,
		Prompt: fullPrompt,
		Format: json.RawMessage(`"json"`),
		Stream: &stream,
		Options: map[string]any{
			"temperature": 0.1, // Low temperature for consistent output
		},
	}, func(resp api.GenerateResponse) error {
		response.WriteString(resp.Response)
		if resp.Done {
			evalCount = resp.EvalCount
			promptEvalCount = resp.PromptEvalCount
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ollama generate: %w", err)
	}

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(response.String()), &aiResp); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w (raw: %s)", err, response.String())
	}

	return &AnalyzeResponse{
		Suggestions: ConvertToSuggestions(aiResp, req.ExistingTags, req.Correspondents),
		Usage: Usage{
			InputTokens:  promptEvalCount,
			OutputTokens: evalCount,
			Model:        p.model,
		},
	}, nil
}
