package ai

import (
	"context"

	"github.com/google/uuid"
)

// Provider abstracts AI provider differences
type Provider interface {
	// Analyze sends document text and returns AI suggestions
	Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error)
	// Name returns the provider identifier (openai, anthropic, ollama)
	Name() string
	// Available checks if the provider is configured (env vars present)
	Available() bool
}

// AnalyzeRequest contains document content and context for AI analysis
type AnalyzeRequest struct {
	DocumentID     uuid.UUID
	TextContent    string   // First N pages of extracted text
	ExistingTags   []string // User's current tag taxonomy
	Correspondents []string // User's current correspondents
	MaxTokens      int      // Max response tokens (default 1024)
}

// AnalyzeResponse contains AI suggestions and usage data
type AnalyzeResponse struct {
	Suggestions []Suggestion
	Usage       Usage
}

// Suggestion represents a single AI recommendation
type Suggestion struct {
	Type       string  // "tag" or "correspondent"
	Value      string  // The suggested name
	Confidence float64 // 0.0 to 1.0
	Reasoning  string  // Brief explanation
	IsNew      bool    // True if not in existing taxonomy
}

// Usage tracks token consumption for cost monitoring
type Usage struct {
	InputTokens  int
	OutputTokens int
	Model        string
}

// AIResponse is the JSON schema for AI responses (all providers)
// Note: correspondent uses pointer with nullable tag for OpenAI structured output compatibility
type AIResponse struct {
	Tags          []TagSuggestion          `json:"tags" jsonschema_description:"Array of suggested tags for the document"`
	Correspondent *CorrespondentSuggestion `json:"correspondent" jsonschema:"nullable" jsonschema_description:"Suggested correspondent (sender/organization), or null if unclear"`
}

// TagSuggestion is the JSON schema for a tag suggestion
type TagSuggestion struct {
	Name       string  `json:"name" jsonschema_description:"The tag name"`
	Confidence float64 `json:"confidence" jsonschema_description:"Confidence score from 0.0 to 1.0"`
	Reasoning  string  `json:"reasoning" jsonschema_description:"Brief explanation for this suggestion"`
}

// CorrespondentSuggestion is the JSON schema for a correspondent suggestion
type CorrespondentSuggestion struct {
	Name       string  `json:"name" jsonschema_description:"The correspondent name"`
	Confidence float64 `json:"confidence" jsonschema_description:"Confidence score from 0.0 to 1.0"`
	Reasoning  string  `json:"reasoning" jsonschema_description:"Brief explanation for this suggestion"`
}

// ConvertToSuggestions converts AIResponse to []Suggestion
func ConvertToSuggestions(resp AIResponse, existingTags, existingCorrespondents []string) []Suggestion {
	var suggestions []Suggestion

	// Convert tag suggestions
	tagSet := make(map[string]bool)
	for _, t := range existingTags {
		tagSet[t] = true
	}
	for _, t := range resp.Tags {
		suggestions = append(suggestions, Suggestion{
			Type:       "tag",
			Value:      t.Name,
			Confidence: t.Confidence,
			Reasoning:  t.Reasoning,
			IsNew:      !tagSet[t.Name],
		})
	}

	// Convert correspondent suggestion
	if resp.Correspondent != nil {
		corrSet := make(map[string]bool)
		for _, c := range existingCorrespondents {
			corrSet[c] = true
		}
		suggestions = append(suggestions, Suggestion{
			Type:       "correspondent",
			Value:      resp.Correspondent.Name,
			Confidence: resp.Correspondent.Confidence,
			Reasoning:  resp.Correspondent.Reasoning,
			IsNew:      !corrSet[resp.Correspondent.Name],
		})
	}

	return suggestions
}
