package ai

import (
	"strings"
)

// SystemPrompt is the system instruction for AI analysis
const SystemPrompt = `You are a document analysis assistant. Analyze the provided document text and suggest:
1. Tags that categorize this document (e.g., invoice, receipt, contract, medical, insurance, tax, bank)
2. The correspondent (sender/recipient organization or person)

IMPORTANT:
- Prefer existing tags/correspondents when they match
- Only suggest new ones if no existing option fits well
- Assign confidence scores (0.0-1.0) based on how certain you are
- Provide brief reasoning for each suggestion
- Suggest 1-5 tags maximum, focusing on the most relevant
- Suggest exactly one correspondent (or omit if unclear)

Your response must be valid JSON matching the schema.`

// BuildPrompt constructs the user prompt with document context
func BuildPrompt(req AnalyzeRequest) string {
	var b strings.Builder

	b.WriteString("## Existing Tags\n")
	if len(req.ExistingTags) > 0 {
		for _, t := range req.ExistingTags {
			b.WriteString("- ")
			b.WriteString(t)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("(none yet)\n")
	}

	b.WriteString("\n## Existing Correspondents\n")
	if len(req.Correspondents) > 0 {
		for _, c := range req.Correspondents {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("(none yet)\n")
	}

	b.WriteString("\n## Document Text (first pages)\n")
	b.WriteString(req.TextContent)

	return b.String()
}

// JSONSchema is the schema for structured output (OpenAI/Anthropic format)
var JSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"tags": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":       map[string]any{"type": "string"},
					"confidence": map[string]any{"type": "number"},
					"reasoning":  map[string]any{"type": "string"},
				},
				"required": []string{"name", "confidence", "reasoning"},
			},
		},
		"correspondent": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":       map[string]any{"type": "string"},
				"confidence": map[string]any{"type": "number"},
				"reasoning":  map[string]any{"type": "string"},
			},
			"required": []string{"name", "confidence", "reasoning"},
		},
	},
	"required": []string{"tags"},
}
