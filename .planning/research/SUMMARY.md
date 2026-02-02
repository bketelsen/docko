# Project Research Summary

**Project:** Docko - PDF Document Management System
**Domain:** Document Management / Self-hosted Archive
**Researched:** 2026-02-02
**Confidence:** MEDIUM-HIGH

## Executive Summary

Docko is a self-hosted PDF document management system for household/small team use at tens of thousands of documents scale. The research indicates this is a well-established domain with proven patterns: PostgreSQL-backed queue processing, full-text search via tsvector, and async processing pipeline. The recommended approach leverages Go's existing stack (Echo + PostgreSQL) extended with pdfcpu for PDF handling, River for queue management, and go-smb2 for network share integration. No new infrastructure dependencies (Redis, Elasticsearch) are needed.

The key differentiator is **native network share integration** (SMB/NFS without OS mounts) combined with **modern LLM-powered auto-tagging**, positioning Docko as more network-aware than Paperless-ngx and simpler than Mayan EDMS. The architecture follows a six-component model with queue-based processing: document store, metadata database, processing pipeline, search subsystem, source watchers, and AI integration layer.

Critical risks center on file handling correctness (atomic writes, race conditions, partial uploads), queue reliability (lost jobs, poison messages), and AI cost control (runaway API spend, rate limiting). These are all mitigated through well-documented patterns: temp-file-then-rename for atomic writes, PostgreSQL-backed queue with visibility timeouts, and strict cost tracking with page limits for AI. The build order must start with foundation (document store + database schema + queue infrastructure) before layering on processing, sources, and AI features.

## Key Findings

### Recommended Stack

The stack extends the existing Go + Echo + PostgreSQL foundation without requiring additional infrastructure. PostgreSQL handles both queuing (via River) and full-text search (via tsvector), eliminating Redis and Elasticsearch complexity.

**Core technologies:**
- **pdfcpu + ledongthuc/pdf** (PDF processing) — pdfcpu for validation/metadata, ledongthuc/pdf for text extraction. Pure Go, Apache/MIT licensed. Avoids unidoc (commercial) and CGO dependencies.
- **River** (queue system) — PostgreSQL-native job queue. Transactional enqueue, no Redis needed. Gained strong adoption in 2024-2025 Go ecosystem.
- **PostgreSQL tsvector + GIN index** (full-text search) — Built-in FTS sufficient for tens of thousands of docs. Generated ALWAYS AS STORED keeps index synchronized automatically.
- **go-smb2** (SMB shares) — Pure Go SMB 2.x/3.x client for network share integration without OS mounts.
- **pdftoppm** (thumbnail generation) — System dependency (poppler-utils) for high-quality PDF rendering. Standard approach; pure Go alternatives insufficient.
- **Direct HTTP clients for AI** (OpenAI/Anthropic/Ollama) — Avoid heavy SDKs, implement provider interface for flexibility. Ollama option for cost-free local inference.

**Anti-recommendations:** unidoc/unipdf (commercial), Elasticsearch (overkill), Redis (unnecessary), CGO-based PDF libraries (deployment complexity), heavy ORMs (project uses sqlc).

### Expected Features

Research analyzed Paperless-ngx, Mayan EDMS, and Docspell to identify table stakes vs differentiators. Users expect core DMS features; AI tagging and network shares are differentiators.

**Must have (table stakes):**
- Web upload with drag-and-drop and bulk upload
- Watch folder / inbox for local directory monitoring
- Tags and correspondents with full CRUD
- Full-text search with filters (tags, correspondent, date range)
- In-browser PDF viewer with thumbnails
- Document download and metadata editing
- Text extraction and page count detection

**Should have (competitive differentiators):**
- **AI-powered auto-tagging** — LLM-based tagging eliminates manual categorization, surpasses traditional ML/rules approaches
- **Network share integration** — SMB/NFS watching without OS mounts, unique to Docko
- **Processing queue dashboard** — Visibility into async pipeline, audit trail per document

**Defer (v2+):**
- OCR (explicitly out of scope in PROJECT.md for v1)
- NFS integration (if SMB proves sufficient)
- Matching rules engine (complex, can start with AI tagging)
- Correspondent auto-detection (can be manual initially)
- Saved searches, bulk operations, boolean operators (nice-to-have)

### Architecture Approach

Document management at this scale follows a six-component architecture with queue-based async processing. PostgreSQL serves triple duty: metadata store, queue backend, and search index. All processing is asynchronous through well-defined queue stages.

**Major components:**
1. **Document Store (File System Layer)** — UUID-named flat directories for originals, thumbnails, temp. Inbox isolation prevents processing partial uploads.
2. **Metadata Database (PostgreSQL)** — Documents, tags, correspondents, queue_jobs, document_audit_log. Separate document_content table keeps main table lean. Generated tsvector column auto-updates for FTS.
3. **Processing Pipeline (Queue-Based Workers)** — Ingestion → Duplicate Check → (Text Extract || Thumbnail) → AI Tagging. River-based workers with independent concurrency per queue type.
4. **Search Subsystem (PostgreSQL Full-Text)** — GIN index on tsvector, ts_rank for relevance, ts_headline for snippets. websearch_to_tsquery for user-friendly queries.
5. **Source Management (Watchers)** — fsnotify for local (event-based), go-smb2 for SMB (polling), optional NFS. File stability check prevents partial upload processing.
6. **AI Integration Layer** — Provider interface (OpenAI/Claude/Ollama) with cost tracking, page limits, and prompt templates. Temperature=0 for consistency.

**Data flow:** Source detects file → stability check → create pending record → copy to originals with UUID → compute SHA-256 hash → duplicate check → parallel text extract + thumbnail → AI tagging (if enabled) → status: ready → audit log.

### Critical Pitfalls

Research identified 40+ pitfalls across file handling, queue processing, search, AI, and network shares. Top 5 by severity and phase timing:

1. **Incomplete file writes leading to corruption (Critical, Phase 1)** — Write to temp file first, verify size, atomic rename. Never process files during copy. Essential for ingestion reliability.

2. **Lost jobs on worker crash (Critical, Phase 1)** — Use visibility timeout pattern with `SELECT ... FOR UPDATE SKIP LOCKED`. Jobs stuck in "processing" must auto-retry after timeout. Foundational to queue infrastructure.

3. **Runaway AI API costs (Critical, Phase 3)** — Track cost per document, limit pages sent to AI (default 3-5), implement daily/monthly spend limits. Test against mock API. Easy to rack up $500/month instead of $50.

4. **Race conditions on concurrent file access (Critical, Phase 1)** — Use PostgreSQL advisory locks keyed on content hash. Idempotent ingestion. File watchers can fire twice for same file on network shares.

5. **Network timeout handling for shares (Critical, Phase 1)** — Set explicit timeouts (30s), retry transient errors with exponential backoff, re-authenticate before each batch. NAS devices sleep, switches reboot, sessions expire.

**Additional high-priority pitfalls:** Storage exhaustion without warning (check space before batch), rate limit handling for AI (exponential backoff, don't count as retry), poor prompt engineering (provide examples, constrain categories), authentication token expiry for SMB (refresh connections proactively), unbounded search index growth (limit indexed content per doc).

## Implications for Roadmap

Based on research, suggested **4-phase structure** organized around dependency chain:

### Phase 1: Foundation & Ingestion
**Rationale:** Everything depends on reliable document storage and queue processing. Must establish data model and async processing before any features.

**Delivers:** Documents can be uploaded via web UI and watched from local folders. Files are stored safely with UUID naming, hashed for duplicate detection, and queued for processing. Queue infrastructure supports retry, timeout, and audit logging.

**Addresses (from FEATURES.md):**
- Web upload with drag-and-drop
- Watch folder / inbox (local)
- Duplicate detection
- Tags and correspondents (schema + CRUD, no auto-detection yet)
- Processing queue infrastructure

**Avoids (from PITFALLS.md):**
- Incomplete file writes (temp-then-rename pattern)
- Race conditions (PostgreSQL advisory locks)
- Lost jobs (visibility timeout pattern)
- Storage exhaustion (space checks, thresholds)
- File handle leaks (defer close, connection pooling)

**Components built:** Document Store (`internal/store/`), Database Schema (migrations), Queue System (`internal/queue/`), Ingestion Pipeline (`internal/processor/ingest.go`, `duplicate.go`), Local Source Watcher (`internal/source/local.go`).

**Stack additions:** River (queue), pdfcpu (validation, metadata).

### Phase 2: Text Extraction & Search
**Rationale:** Search is the core value proposition. Depends on ingestion pipeline to have documents. Text extraction enables both search and AI features (Phase 3).

**Delivers:** Documents are searchable by full content and metadata. Users can filter by tags, correspondent, date range. Search results show snippets with highlighted matches. Thumbnails provide visual navigation.

**Addresses (from FEATURES.md):**
- Full-text search (PostgreSQL tsvector)
- Filter by tags, correspondent, date range
- Search results preview with snippets
- Thumbnail generation
- In-browser PDF viewer
- Document download

**Uses (from STACK.md):**
- PostgreSQL tsvector + GIN index
- ledongthuc/pdf (text extraction)
- pdftoppm (thumbnail generation via system exec)
- ts_rank, ts_headline for relevance and snippets

**Implements (from ARCHITECTURE.md):** Search Subsystem, Text Extraction Worker, Thumbnail Worker. Search service (`internal/search/`) wraps PostgreSQL FTS queries with faceted filtering. Document Management UI (Templ templates + HTMX) for listing, detail view, tag management.

**Avoids (from PITFALLS.md):**
- Unbounded index growth (limit indexed content to first 100KB)
- Poor search relevance (use ts_rank weights, title > content)
- Search query injection (websearch_to_tsquery instead of to_tsquery)
- Memory bloat during indexing (stream page-by-page, limit content)

### Phase 3: Network Shares & AI Tagging
**Rationale:** Key differentiators but independent of each other. Both depend on Phases 1-2 being solid. Can be built in parallel or sequenced based on priority.

**Delivers:** Documents auto-import from SMB shares without OS mounts. AI suggests tags based on document content, reducing manual categorization work. Both features have cost controls and failure resilience.

**Addresses (from FEATURES.md):**
- SMB share watching (go-smb2)
- AI-powered auto-tagging (OpenAI/Claude/Ollama)
- Tag suggestions (AI assigns with confidence scores)
- Processing pipeline visibility (dashboard, audit trail)

**Uses (from STACK.md):**
- go-smb2 for SMB client
- Direct HTTP clients for AI providers
- Provider interface for flexibility

**Implements (from ARCHITECTURE.md):** Network Sources (`internal/source/smb.go`), AI Provider Interface (`internal/ai/provider.go`, `openai.go`, `ollama.go`), AI Processing Workers (tag suggestion, configurable prompt templates).

**Avoids (from PITFALLS.md):**
- Runaway AI costs (track spend, limit pages, daily/monthly caps)
- Rate limit handling (exponential backoff, circuit breaker, low concurrency)
- Poor prompt engineering (provide examples, constrain categories, temperature=0)
- Authentication token expiry (reconnect before batch, health checks)
- Network timeout handling (explicit timeouts, retry transient errors)
- File locking conflicts on shares (copy to local temp first)

### Phase 4: Polish & Optimization
**Rationale:** After MVP is functional, improve UX and operational visibility. These features don't block core value delivery.

**Delivers:** Improved user experience with keyboard shortcuts, bulk operations, saved searches. Admin dashboard for queue health, processing statistics. Documentation for deployment and operation.

**Addresses (from FEATURES.md):**
- Keyboard navigation (j/k shortcuts)
- Bulk operations (multi-select, apply action)
- Saved searches (store query parameters)
- Dashboard overview (stats, recent activity)
- Processing statistics (counters, averages)

**Avoids (from PITFALLS.md):**
- Missing health checks (database, queue depth, share connectivity)
- No audit trail (already addressed in Phase 1, but enhance UI)

### Phase Ordering Rationale

- **Dependencies dictate order:** Can't search without text extraction (Phase 2 depends on Phase 1). Can't do AI tagging without text (Phase 3 depends on Phase 2). Foundation must be rock-solid before adding features.
- **Risk mitigation by isolation:** File handling and queue pitfalls (highest severity) addressed in Phase 1 before building on top. Search complexity isolated in Phase 2. Expensive/complex features (AI, network shares) deferred to Phase 3 when foundation is proven.
- **Parallel vs sequential:** Phases 1-2 are sequential. Phase 3 components (network shares + AI) can be built in parallel if resources allow. Phase 4 is ongoing polish.
- **MVP scope:** Phases 1-2 deliver a functional document archive with search. Phase 3 adds differentiators. Phase 4 is post-MVP refinement.

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 1 (Network Shares):** go-smb2 library specifics, error handling patterns, performance characteristics need verification during implementation. Confidence: MEDIUM-LOW.
- **Phase 3 (AI Integration):** Prompt engineering is iterative; will need real-world testing with actual documents. Provider API changes may require adjustments. Confidence: MEDIUM.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Ingestion, Queue, Duplicate Detection):** Well-documented patterns, PostgreSQL advisory locks standard, SHA-256 hashing proven. Confidence: MEDIUM-HIGH.
- **Phase 2 (Full-Text Search):** PostgreSQL FTS is mature, extensive documentation available. Patterns are well-established. Confidence: HIGH.
- **Phase 2 (Text Extraction):** PDF libraries have standard APIs, text extraction well-documented. Confidence: MEDIUM.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM | Training data through May 2025; versions need verification. PostgreSQL FTS patterns HIGH confidence, River/pdfcpu MEDIUM (verify current versions). |
| Features | MEDIUM | Based on analysis of Paperless-ngx, Mayan EDMS, Docspell (mature projects). Table stakes well-established; differentiator value is assumption. |
| Architecture | HIGH | Document management is well-trodden domain with proven patterns. Queue-based processing, PostgreSQL FTS, UUID naming all standard. |
| Pitfalls | MEDIUM | File handling and queue pitfalls HIGH confidence (general systems knowledge). SMB/NFS specifics MEDIUM-LOW (library-dependent, need verification). AI pitfalls MEDIUM (general patterns, provider specifics vary). |

**Overall confidence:** MEDIUM-HIGH

Research provides strong foundation for roadmap decisions. Architecture patterns are proven. Stack choices are sound given constraints (no new infrastructure). Pitfall mitigation strategies are well-documented.

### Gaps to Address

Research was conducted without web access; all recommendations based on training data through May 2025. These gaps should be addressed:

- **Verify library versions before adding dependencies:** Run `go list -m -versions [repo]` for pdfcpu, River, go-smb2, ledongthuc/pdf, AI SDKs. Versions may have changed significantly since May 2025.
- **Test go-smb2 with target NAS devices:** SMB library compatibility varies by NAS vendor (Synology, QNAP, etc.). Early testing recommended to surface issues before full implementation.
- **Validate AI provider SDK status:** Anthropic/OpenAI SDK landscape changes rapidly. During Phase 3 planning, verify current SDK recommendations or confirm direct HTTP approach.
- **Benchmark PostgreSQL FTS at scale:** Confidence is high for "tens of thousands" scale, but actual query performance should be validated with representative data during Phase 2.
- **poppler-utils availability:** Confirm pdftoppm is available/installable on target deployment platform (Docker base image, server OS). Fallback strategy if unavailable.

## Sources

### Primary (HIGH confidence)
- **PostgreSQL documentation** — Full-text search (tsvector, GIN index, ts_rank, ts_headline), advisory locks, generated columns. Well-established features.
- **General systems knowledge** — File handling patterns (atomic writes, temp-then-rename), queue processing (visibility timeout, at-least-once delivery), SHA-256 hashing for deduplication.

### Secondary (MEDIUM confidence)
- **Go ecosystem training data (through May 2025)** — River adoption in 2024-2025, pdfcpu as mature PDF library, go-smb2 as common SMB client choice, ledongthuc/pdf for text extraction.
- **Document management domain patterns** — Analysis of Paperless-ngx, Mayan EDMS, Docspell architectures and feature sets. These are open-source references, but feature sets may have evolved.

### Tertiary (LOW confidence, needs validation)
- **AI SDK recommendations** — Anthropic SDK status, OpenAI community SDKs. This area changes rapidly; direct HTTP may be safer bet but verify during Phase 3 planning.
- **go-smb2 specifics** — Error handling patterns, compatibility with specific NAS vendors, performance characteristics. Needs hands-on verification during implementation.

---
*Research completed: 2026-02-02*
*Ready for roadmap: yes*
