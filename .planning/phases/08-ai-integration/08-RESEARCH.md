# Phase 8: AI Integration - Research

**Researched:** 2026-02-03
**Domain:** LLM API integration for document tagging and correspondent detection
**Confidence:** HIGH

## Summary

This phase integrates AI providers (OpenAI, Claude, Ollama) for automated document tagging and correspondent detection. The research confirms that official Go SDKs exist for all three providers, with consistent patterns for API interaction, structured JSON output, and usage tracking.

The standard approach uses a provider abstraction layer that wraps the three official SDKs, enabling fallback between providers while maintaining a unified interface. Structured outputs are now standard across all major providers, with JSON schema enforcement ensuring reliable parsing of AI suggestions.

Key findings:
- Official Go SDKs: `openai-go v3.x`, `anthropic-sdk-go v1.x`, `ollama/api`
- All providers support structured JSON output with schema validation
- Token counting available in API responses for cost tracking
- Confidence thresholds of 0.8 (auto-apply) and 0.5 (review) are industry standard

**Primary recommendation:** Build a thin provider interface around the three official SDKs. Use structured JSON output with a consistent schema for tag/correspondent suggestions including confidence scores.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/openai/openai-go/v3 | v3.17.0 | OpenAI API client | Official SDK, maintained by OpenAI |
| github.com/anthropics/anthropic-sdk-go | v1.20.0 | Anthropic Claude API client | Official SDK, maintained by Anthropic |
| github.com/ollama/ollama/api | latest | Ollama local model client | Official package used by Ollama CLI |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | JSON parsing | Parse AI responses, build prompts |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Individual SDKs | github.com/dariubs/goai | Unified interface but less control, potential lag behind official SDK updates |
| Individual SDKs | github.com/manishiitg/multi-llm-provider-go | Multi-provider but additional dependency |

**Installation:**
```bash
go get github.com/openai/openai-go/v3@v3.17.0
go get github.com/anthropics/anthropic-sdk-go@v1.20.0
# Ollama API is included with ollama installation, or:
go get github.com/ollama/ollama/api
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── ai/
│   ├── ai.go            # Provider interface, types, factory
│   ├── openai.go        # OpenAI provider implementation
│   ├── anthropic.go     # Anthropic provider implementation
│   ├── ollama.go        # Ollama provider implementation
│   ├── prompt.go        # Prompt templates and building
│   └── service.go       # AI service orchestration, queue integration
├── database/
│   └── sqlc/            # Add ai_suggestions, ai_settings, ai_usage tables
└── handler/
    └── ai.go            # AI settings, queue status, review endpoints
```

### Pattern 1: Provider Interface
**What:** Abstract provider differences behind a common interface
**When to use:** Always - enables provider switching and fallback
**Example:**
```go
// Source: Based on GoAI and multi-llm-provider-go patterns
type Provider interface {
    // Analyze sends document text and returns AI suggestions
    Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error)
    // Name returns the provider identifier (openai, anthropic, ollama)
    Name() string
    // Available checks if the provider is configured and reachable
    Available() bool
}

type AnalyzeRequest struct {
    DocumentID    uuid.UUID
    TextContent   string      // First N pages of extracted text
    ExistingTags  []string    // User's current tag taxonomy
    Correspondents []string   // User's current correspondents
    MaxTokens     int
}

type AnalyzeResponse struct {
    Suggestions []Suggestion
    Usage       Usage
}

type Suggestion struct {
    Type       string  // "tag" or "correspondent"
    Value      string  // The suggested tag name or correspondent name
    Confidence float64 // 0.0 to 1.0
    Reasoning  string  // Brief explanation for UI display
    IsNew      bool    // True if not in existing taxonomy
}

type Usage struct {
    InputTokens  int
    OutputTokens int
    Model        string
}
```

### Pattern 2: Structured JSON Output
**What:** Define JSON schema for consistent AI responses
**When to use:** All AI requests - ensures reliable parsing
**Example:**
```go
// Schema for AI response - all providers support JSON output
// OpenAI: response_format with json_schema
// Anthropic: output_config.format with json_schema
// Ollama: format: "json" in ChatRequest
type AIResponse struct {
    Tags []TagSuggestion `json:"tags"`
    Correspondent *CorrespondentSuggestion `json:"correspondent"`
}

type TagSuggestion struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`
    Reasoning  string  `json:"reasoning"`
}

type CorrespondentSuggestion struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`
    Reasoning  string  `json:"reasoning"`
}
```

### Pattern 3: Provider Fallback Chain
**What:** Try providers in order, falling back on failure
**When to use:** When multiple providers are configured
**Example:**
```go
// Source: Industry standard pattern from LLM gateways
func (s *Service) analyzeWithFallback(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
    providers := s.getOrderedProviders() // Preferred first

    var lastErr error
    for _, p := range providers {
        if !p.Available() {
            continue
        }

        resp, err := p.Analyze(ctx, req)
        if err != nil {
            lastErr = err
            slog.Warn("provider failed, trying next",
                "provider", p.Name(),
                "error", err)
            continue
        }
        return resp, nil
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
```

### Pattern 4: Suggestion Storage and State Machine
**What:** Store AI suggestions with status for review workflow
**When to use:** All suggestions - enables accept/reject flow
**Example:**
```sql
-- ai_suggestions table
CREATE TYPE suggestion_status AS ENUM ('pending', 'accepted', 'rejected', 'auto_applied');

CREATE TABLE ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id),
    suggestion_type VARCHAR(20) NOT NULL, -- 'tag' or 'correspondent'
    value VARCHAR(255) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL, -- 0.00 to 1.00
    reasoning TEXT,
    status suggestion_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(50) -- 'auto' or 'user'
);
```

### Anti-Patterns to Avoid
- **Direct provider coupling:** Never call provider SDKs directly from handlers. Always go through the service layer with the provider interface.
- **Blocking on AI calls:** Always process AI analysis through the job queue. Never make synchronous AI calls in HTTP handlers.
- **Ignoring rate limits:** Always implement exponential backoff. Providers return 429 errors on rate limiting.
- **Unbounded text input:** Always truncate to max pages. Sending full documents wastes tokens and hits context limits.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token counting | Character-based estimation | API response usage field | Each provider/model uses different tokenizers |
| JSON schema validation | Manual parsing with regex | Structured output feature | SDKs guarantee schema compliance |
| Rate limiting | Simple sleep | Exponential backoff with jitter | Already implemented in queue system |
| Retry logic | Custom retry code | Queue's existing retry mechanism | Consistent retry behavior across all job types |

**Key insight:** All three providers now support structured JSON output natively. This eliminates the need for prompt engineering to force JSON format or post-processing to extract data from free-form responses.

## Common Pitfalls

### Pitfall 1: Assuming Token = Character
**What goes wrong:** Underestimating token usage leads to unexpected costs
**Why it happens:** Developers assume 1 token = 1 word or 4 characters
**How to avoid:** Use the usage field returned by all providers; store and track actual tokens
**Warning signs:** Cost estimates don't match actual bills

### Pitfall 2: Not Handling Partial Responses
**What goes wrong:** AI response cut off mid-JSON due to max_tokens
**Why it happens:** Complex documents generate long responses
**How to avoid:** Set appropriate max_tokens (1024 sufficient for tag suggestions), check stop_reason
**Warning signs:** JSON parse errors, stop_reason = "max_tokens"

### Pitfall 3: Ignoring Provider Differences in Confidence
**What goes wrong:** Different providers return different confidence distributions
**Why it happens:** Models are trained differently, confidence calibration varies
**How to avoid:** Test thresholds per provider, consider normalizing or using provider-specific thresholds
**Warning signs:** One provider auto-applies everything, another requires review for everything

### Pitfall 4: Not Including Taxonomy in Prompts
**What goes wrong:** AI suggests tags/correspondents that don't match user's naming conventions
**Why it happens:** Without context, AI invents its own taxonomy
**How to avoid:** Always include existing tags and correspondents in the prompt
**Warning signs:** "invoice" vs "Invoices", "John Smith" vs "Smith, John"

### Pitfall 5: Synchronous AI Processing
**What goes wrong:** HTTP requests timeout waiting for AI response
**Why it happens:** AI API calls take 2-10+ seconds
**How to avoid:** Always use job queue for AI processing, use SSE for status updates
**Warning signs:** Timeouts, slow page loads, blocking UI

## Code Examples

Verified patterns from official sources:

### OpenAI Chat Completion with Structured Output
```go
// Source: https://github.com/openai/openai-go
import (
    "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
)

func (p *OpenAIProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
    client := openai.NewClient(
        option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    prompt := buildPrompt(req) // Include existing tags/correspondents

    chatResp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: openai.ChatModelGPT4oMini,
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(systemPrompt),
            openai.UserMessage(prompt),
        },
        ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
            openai.ResponseFormatJSONSchemaParam{
                Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
                JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
                    Name:   openai.F("document_analysis"),
                    Schema: openai.F(analysisJSONSchema),
                    Strict: openai.Bool(true),
                }),
            },
        ),
        MaxTokens: openai.Int(1024),
    })
    if err != nil {
        return nil, fmt.Errorf("openai completion: %w", err)
    }

    var aiResp AIResponse
    if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &aiResp); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    return &AnalyzeResponse{
        Suggestions: convertSuggestions(aiResp),
        Usage: Usage{
            InputTokens:  int(chatResp.Usage.PromptTokens),
            OutputTokens: int(chatResp.Usage.CompletionTokens),
            Model:        string(chatResp.Model),
        },
    }, nil
}
```

### Anthropic Claude with JSON Output
```go
// Source: https://github.com/anthropics/anthropic-sdk-go
import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

func (p *AnthropicProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
    client := anthropic.NewClient(
        option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    )

    prompt := buildPrompt(req)

    message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.ModelClaudeHaiku4_5, // Cost-effective for tagging
        MaxTokens: 1024,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
        },
        System: anthropic.F([]anthropic.TextBlockParam{
            anthropic.NewTextBlock(systemPrompt),
        }),
        OutputConfig: anthropic.F(anthropic.OutputConfigParam{
            Format: anthropic.F[anthropic.OutputConfigFormatUnionParam](
                anthropic.OutputConfigJSONSchemaParam{
                    Type:   anthropic.F(anthropic.OutputConfigJSONSchemaTypeJSONSchema),
                    Schema: anthropic.F(analysisJSONSchema),
                },
            ),
        }),
    })
    if err != nil {
        return nil, fmt.Errorf("anthropic message: %w", err)
    }

    // Response is in message.Content[0].Text for JSON output
    var aiResp AIResponse
    if err := json.Unmarshal([]byte(message.Content[0].Text), &aiResp); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    return &AnalyzeResponse{
        Suggestions: convertSuggestions(aiResp),
        Usage: Usage{
            InputTokens:  int(message.Usage.InputTokens),
            OutputTokens: int(message.Usage.OutputTokens),
            Model:        string(message.Model),
        },
    }, nil
}
```

### Ollama Local Model
```go
// Source: https://pkg.go.dev/github.com/ollama/ollama/api
import "github.com/ollama/ollama/api"

func (p *OllamaProvider) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalyzeResponse, error) {
    client, err := api.ClientFromEnvironment()
    if err != nil {
        return nil, fmt.Errorf("create ollama client: %w", err)
    }

    prompt := buildPrompt(req)
    model := os.Getenv("OLLAMA_MODEL")
    if model == "" {
        model = "llama3.2" // Default
    }

    var response strings.Builder
    var metrics api.Metrics

    err = client.Chat(ctx, &api.ChatRequest{
        Model: model,
        Messages: []api.Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: prompt},
        },
        Format: json.RawMessage(`"json"`), // Force JSON output
        Stream: boolPtr(false),
    }, func(resp api.ChatResponse) error {
        response.WriteString(resp.Message.Content)
        if resp.Done {
            metrics = resp.Metrics
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("ollama chat: %w", err)
    }

    var aiResp AIResponse
    if err := json.Unmarshal([]byte(response.String()), &aiResp); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    return &AnalyzeResponse{
        Suggestions: convertSuggestions(aiResp),
        Usage: Usage{
            InputTokens:  metrics.PromptEvalCount,
            OutputTokens: metrics.EvalCount,
            Model:        model,
        },
    }, nil
}
```

### System Prompt Template
```go
const systemPrompt = `You are a document analysis assistant. Analyze the provided document text and suggest:
1. Tags that categorize this document (e.g., invoice, receipt, contract, medical, insurance)
2. The correspondent (sender/recipient organization or person)

IMPORTANT:
- Prefer existing tags/correspondents when they match
- Only suggest new ones if no existing option fits
- Assign confidence scores (0.0-1.0) based on how certain you are
- Provide brief reasoning for each suggestion

Output format is strictly enforced by JSON schema.`

func buildPrompt(req AnalyzeRequest) string {
    var b strings.Builder

    b.WriteString("## Existing Tags\n")
    if len(req.ExistingTags) > 0 {
        for _, t := range req.ExistingTags {
            b.WriteString("- " + t + "\n")
        }
    } else {
        b.WriteString("(none yet)\n")
    }

    b.WriteString("\n## Existing Correspondents\n")
    if len(req.Correspondents) > 0 {
        for _, c := range req.Correspondents {
            b.WriteString("- " + c + "\n")
        }
    } else {
        b.WriteString("(none yet)\n")
    }

    b.WriteString("\n## Document Text (first pages)\n")
    b.WriteString(req.TextContent)

    return b.String()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Prompt engineering for JSON | Structured outputs with schema | 2025-2026 | Guaranteed valid JSON, no retries needed |
| Temperature 0 for consistency | Temperature 0 + structured output | 2025 | More reliable, still creative within schema |
| Community Go SDKs | Official Go SDKs | 2025-2026 | Better support, faster updates |
| Function calling for extraction | JSON schema output | 2025-2026 | Simpler for pure extraction (no function execution needed) |

**Deprecated/outdated:**
- `sashabaranov/go-openai`: Community library, now superseded by official `openai-go`
- Beta headers for structured output: Anthropic's `anthropic-beta: structured-outputs-2025-11-13` no longer required
- JSON mode without schema: Raw `json` format less reliable than schema-constrained output

## Confidence Threshold Recommendations

Based on industry research and the project's requirement for high-confidence auto-apply:

| Threshold | Action | Rationale |
|-----------|--------|-----------|
| >= 0.85 | Auto-apply | High precision, minimal false positives |
| 0.50 - 0.84 | Pending review | Reasonable suggestions, user verification adds value |
| < 0.50 | Discard | Too uncertain, not worth user's review time |

**Implementation notes:**
- Show top 3-5 suggestions per document (not all)
- Store all suggestions above 0.50 for audit trail
- Consider per-provider calibration if confidence distributions differ significantly

## Open Questions

Things that couldn't be fully resolved:

1. **Ollama JSON schema strictness**
   - What we know: Ollama supports `format: "json"` for JSON output
   - What's unclear: Whether schema enforcement is as strict as OpenAI/Anthropic
   - Recommendation: Test thoroughly, may need prompt-based schema enforcement as fallback

2. **Token estimation before request**
   - What we know: Each provider uses different tokenizers
   - What's unclear: No unified Go library for pre-request token estimation
   - Recommendation: Use conservative page limits (5 pages = ~2000-3000 tokens typically); track actual usage and adjust

3. **Confidence score calibration across providers**
   - What we know: Different models produce different confidence distributions
   - What's unclear: How much calibration is needed
   - Recommendation: Start with same thresholds, monitor acceptance rates, adjust per provider if needed

## Sources

### Primary (HIGH confidence)
- [openai/openai-go](https://github.com/openai/openai-go) - Official Go SDK, v3.17.0, installation and usage patterns
- [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) - Official Go SDK, v1.20.0, structured outputs
- [ollama/ollama/api](https://pkg.go.dev/github.com/ollama/ollama/api) - Official Go package, Chat API, Metrics
- [Anthropic Structured Outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) - JSON schema output, tool use with strict mode

### Secondary (MEDIUM confidence)
- [GoAI multi-provider library](https://dev.to/dariubs/goai-a-clean-multi-provider-llm-client-for-go-27o5) - Interface design patterns
- [LLM Gateway architecture](https://medium.com/@yadav.navya1601/what-is-an-llm-gateway-understanding-the-infrastructure-layer-for-multi-model-ai-fea4fecbc931) - Fallback and abstraction patterns
- [Mindee confidence scores guide](https://www.mindee.com/blog/how-use-confidence-scores-ml-models) - Threshold recommendations

### Tertiary (LOW confidence)
- WebSearch results on token counting - Various approaches, no authoritative Go solution
- Confidence threshold specific values - Industry conventions vary, need validation with actual data

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official SDKs verified, versions confirmed
- Architecture: HIGH - Patterns match existing codebase (queue, SSE, handlers)
- Provider integration: MEDIUM - Structured output patterns from docs, Go-specific examples synthesized
- Confidence thresholds: MEDIUM - Industry conventions, may need calibration

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable domain)
