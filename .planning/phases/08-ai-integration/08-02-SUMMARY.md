---
phase: 08-ai-integration
plan: 02
subsystem: ai
tags: [openai, anthropic, ollama, llm, provider-pattern, structured-output]

# Dependency graph
requires:
  - phase: 08-01
    provides: AI database schema and sqlc queries
provides:
  - Provider interface abstracting AI provider differences
  - OpenAI provider with GPT-4o-mini and structured JSON output
  - Anthropic provider with Claude Haiku 4.5
  - Ollama provider for local LLM inference
  - Prompt builder with system instructions and JSON schema
affects: [08-03-ai-service, 08-04-ai-handlers]

# Tech tracking
tech-stack:
  added:
    - github.com/openai/openai-go v1.12.0
    - github.com/anthropics/anthropic-sdk-go v1.20.0
    - github.com/ollama/ollama v0.15.4
  patterns:
    - Provider interface for AI abstraction
    - Structured JSON output with schema validation
    - Environment-based provider availability

key-files:
  created:
    - internal/ai/ai.go
    - internal/ai/prompt.go
    - internal/ai/openai.go
    - internal/ai/anthropic.go
    - internal/ai/ollama.go

key-decisions:
  - "GPT-4o-mini for OpenAI (cost-effective for tagging)"
  - "Claude Haiku 4.5 for Anthropic (fastest/cheapest Claude)"
  - "llama3.2 default for Ollama (configurable via OLLAMA_MODEL)"
  - "Structured JSON output via OpenAI schema, prompt instructions for others"

patterns-established:
  - "Provider interface: Analyze(ctx, req) + Name() + Available()"
  - "AnalyzeRequest carries document text and existing taxonomy"
  - "Suggestion type with confidence scores and IsNew flag"
  - "Usage tracking for cost monitoring (input/output tokens)"

# Metrics
duration: 6min
completed: 2026-02-03
---

# Phase 8 Plan 2: AI Providers Summary

**Provider interface with OpenAI, Anthropic, and Ollama implementations using official Go SDKs and structured JSON output**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-03T19:41:37Z
- **Completed:** 2026-02-03T19:47:54Z
- **Tasks:** 5
- **Files modified:** 5

## Accomplishments
- Provider interface abstracting AI provider differences with Analyze/Name/Available methods
- OpenAI provider using structured JSON output with GPT-4o-mini model
- Anthropic provider using Claude Haiku 4.5 with JSON instruction in prompt
- Ollama provider for local inference with llama3.2 default model
- Prompt builder including existing tags/correspondents for context-aware suggestions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create provider interface and types** - `e1c90a5` (feat)
2. **Task 2: Create prompt template builder** - `4321b65` (feat)
3. **Task 3: Create OpenAI provider implementation** - `7712576` (feat)
4. **Task 4: Create Anthropic provider implementation** - `b942f7e` (feat)
5. **Task 5: Create Ollama provider implementation** - `37f91fa` (feat)

## Files Created/Modified
- `internal/ai/ai.go` - Provider interface, request/response types, Suggestion with confidence
- `internal/ai/prompt.go` - SystemPrompt constant, BuildPrompt function, JSONSchema definition
- `internal/ai/openai.go` - OpenAI provider with structured output mode
- `internal/ai/anthropic.go` - Anthropic provider with JSON instruction in prompt
- `internal/ai/ollama.go` - Ollama provider with JSON format flag

## Decisions Made
- **OpenAI SDK v1.12.0** - Used latest official Go SDK with structured JSON output support
- **Anthropic SDK v1.20.0** - Used official Go SDK, JSON requested via prompt (no structured output mode)
- **Ollama v0.15.4** - Used official api package with JSON format flag
- **GPT-4o-mini for OpenAI** - Cost-effective model suitable for document tagging tasks
- **Claude Haiku 4.5 for Anthropic** - Fastest/cheapest Claude model for tagging
- **llama3.2 default for Ollama** - Good balance of capability and speed for local inference
- **Low temperature (0.1)** - Consistent JSON output from Ollama
- **ResponseFormatUnion wrapper** - OpenAI SDK requires union type for response format

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Adjusted OpenAI SDK import path**
- **Found during:** Task 3 (OpenAI provider implementation)
- **Issue:** Plan specified `github.com/openai/openai-go/v3` but SDK is at `github.com/openai/openai-go`
- **Fix:** Used correct import path without v3 suffix
- **Files modified:** internal/ai/openai.go
- **Verification:** go build succeeds
- **Committed in:** 7712576

**2. [Rule 1 - Bug] Fixed ResponseFormat union type usage**
- **Found during:** Task 3 (OpenAI provider implementation)
- **Issue:** ResponseFormatJSONSchemaParam cannot be used directly, needs ChatCompletionNewParamsResponseFormatUnion wrapper
- **Fix:** Wrapped schema param in union type with OfJSONSchema pointer field
- **Files modified:** internal/ai/openai.go
- **Verification:** go build succeeds
- **Committed in:** 7712576

**3. [Rule 3 - Blocking] Adjusted Anthropic model constant**
- **Found during:** Task 4 (Anthropic provider implementation)
- **Issue:** SDK uses string for model, not typed constant for Haiku 4.5
- **Fix:** Used model string directly: "claude-haiku-4-5-20251101"
- **Files modified:** internal/ai/anthropic.go
- **Verification:** go build succeeds
- **Committed in:** b942f7e

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All auto-fixes necessary for compilation. SDK APIs differ from plan expectations.

## Issues Encountered
None - SDK documentation was clear once correct import paths identified.

## User Setup Required
None - no external service configuration required. API keys are already expected as environment variables (OPENAI_API_KEY, ANTHROPIC_API_KEY, OLLAMA_HOST).

## Next Phase Readiness
- Provider interface ready for AI service integration (08-03)
- All three providers implement same interface for easy switching
- Usage tracking in place for cost monitoring feature

---
*Phase: 08-ai-integration*
*Completed: 2026-02-03*
