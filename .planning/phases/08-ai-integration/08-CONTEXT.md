# Phase 8: AI Integration - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Automate tagging and correspondent detection using AI providers (OpenAI, Claude, Ollama). Users configure their provider, control costs via page limits, and monitor AI processing through a queue dashboard. AI suggestions integrate with existing organization features from Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Suggestion workflow
- Confidence threshold determines auto-apply vs review
  - High-confidence suggestions auto-apply immediately
  - Low-confidence suggestions require user confirmation
- Pending suggestions visible in two places:
  - Document detail page (inline accept/reject)
  - Dedicated review queue page (batch processing)
- Show top 3-5 most confident suggestions per document (not all)
- "Re-analyze" button always available on document detail
- Include existing tags/correspondents in AI prompt for context
- AI can suggest from existing taxonomy OR propose new ones
- New tag/correspondent auto-created when user accepts the suggestion

### Provider configuration
- API keys via environment variables only (OPENAI_API_KEY, ANTHROPIC_API_KEY)
- OLLAMA_MODEL env var for local model selection
- Admin setting to choose preferred provider (when multiple configured)
- Falls back to next available provider if preferred fails
- On provider failure: retry with exponential backoff, mark failed if retries exhausted

### Cost controls
- Max pages setting: send first N pages only to AI
- Default max pages: 5
- AI processing toggle: global setting, default OFF
  - When ON: new documents auto-process
  - When OFF: manual trigger only
- Track and display detailed usage stats:
  - Documents processed count
  - Tokens used
  - Estimated cost

### Queue monitoring
- Full job history with pagination and status filtering
- Failed jobs show full debug info:
  - Error message
  - Retry count and last attempt time
  - Request/response details for troubleshooting
- Retry controls:
  - Per-job retry button
  - "Retry all failed" bulk action
- Real-time updates via SSE (consistent with Phase 3 processing status)

### Claude's Discretion
- Exact confidence threshold values
- AI prompt engineering and structure
- Token estimation logic per provider
- Job history pagination size
- Debug info formatting

</decisions>

<specifics>
## Specific Ideas

- Include existing tags/correspondents in AI prompts so suggestions match user's taxonomy
- Review queue enables efficient batch processing of low-confidence suggestions
- Detailed usage stats give visibility into AI costs without needing to check provider dashboards

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-ai-integration*
*Context gathered: 2026-02-03*
