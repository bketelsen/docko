# Roadmap: Docko

## Overview

Docko delivers a PDF document management system in 8 phases, progressing from foundation through ingestion, processing, organization, search, and finally AI-powered automation. The journey starts with reliable document storage and queue infrastructure, then builds ingestion pipelines, text extraction, and viewing capabilities. Organization (tags/correspondents) and search follow, enabling the core value of "find any document instantly." Network shares and AI tagging complete the system, delivering the automation that eliminates manual filing.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Document store structure and queue infrastructure
- [x] **Phase 2: Ingestion** - Web upload and local inbox watching
- [x] **Phase 3: Processing** - Text extraction and thumbnail generation
- [x] **Phase 4: Viewing** - PDF preview and download
- [x] **Phase 5: Organization** - Tags and correspondents management
- [x] **Phase 6: Search** - Full-text search with filtering
- [x] **Phase 7: Network Sources** - SMB and NFS share integration
- [x] **Phase 8: AI Integration** - Auto-tagging and correspondent detection
- [ ] **Phase 9: Minimum Number of Words Import Block** - Block document import if text content below threshold
- [ ] **Phase 10: Refactor to Use More templUI Components** - Replace custom UI with templUI components
- [ ] **Phase 11: Dashboard** - Real dashboard at root with stats, counts, and navigation links
- [ ] **Phase 12: Queues Detail** - Queues route with expanders for failed jobs and recent activity

## Phase Details

### Phase 1: Foundation
**Goal**: Establish reliable document storage and queue processing infrastructure
**Depends on**: Nothing (first phase)
**Requirements**: STORE-01, STORE-02, STORE-03, QUEUE-01, QUEUE-04
**Success Criteria** (what must be TRUE):
  1. Documents can be stored with UUID naming in organized directory structure
  2. Original files are preserved unmodified in originals/ directory
  3. Document metadata (filename, size, page count) persists in database
  4. Queue system can accept jobs and process them with retry on failure
  5. Every document processing step is logged in audit trail
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md — Database schema and storage service
- [x] 01-02-PLAN.md — Job queue implementation
- [x] 01-03-PLAN.md — Document service and integration

### Phase 2: Ingestion
**Goal**: Users can add documents via web UI and automated local inbox
**Depends on**: Phase 1
**Requirements**: INGEST-01, INGEST-02, INGEST-03, INGEST-06, INGEST-07
**Success Criteria** (what must be TRUE):
  1. User can drag-and-drop PDF files to upload via web UI
  2. User can upload multiple files at once (bulk upload)
  3. System automatically detects and imports PDFs from local inbox directory
  4. Duplicate documents are detected by content hash before storage
  5. User can configure duplicate handling (delete, rename, skip) per source
**Plans**: 5 plans

Plans:
- [x] 02-01-PLAN.md — Upload handler with PDF validation and dependencies
- [x] 02-02-PLAN.md — Upload UI with drag-drop, progress, and toasts
- [x] 02-03-PLAN.md — Inbox database schema and configuration
- [x] 02-04-PLAN.md — Inbox watcher service with fsnotify
- [x] 02-05-PLAN.md — Wire everything and inbox management UI

### Phase 3: Processing
**Goal**: Uploaded documents are processed for text content and thumbnails
**Depends on**: Phase 2
**Requirements**: QUEUE-02, VIEW-03
**Success Criteria** (what must be TRUE):
  1. Text is extracted from PDFs and indexed in database for search
  2. Thumbnail (first page preview) is generated for each document
  3. Processing happens asynchronously via queue (does not block upload)
**Plans**: 5 plans

Plans:
- [x] 03-01-PLAN.md — Database schema and Docker setup for processing
- [x] 03-02-PLAN.md — Text extraction with embedded + OCR fallback
- [x] 03-03-PLAN.md — Thumbnail generation with WebP output
- [x] 03-04-PLAN.md — Processing job handler and queue wiring
- [x] 03-05-PLAN.md — Status UI with SSE live updates

### Phase 4: Viewing
**Goal**: Users can view and download documents
**Depends on**: Phase 3
**Requirements**: VIEW-01, VIEW-02
**Success Criteria** (what must be TRUE):
  1. User can view PDF in browser without downloading
  2. User can download the original PDF file
  3. Document detail page shows metadata and thumbnail
**Plans**: 3 plans

Plans:
- [x] 04-01-PLAN.md — File serving handlers and templUI components
- [x] 04-02-PLAN.md — Document detail page with tabs and breadcrumbs
- [x] 04-03-PLAN.md — PDF viewer modal with JavaScript controls

### Phase 5: Organization
**Goal**: Users can organize documents with tags and correspondents
**Depends on**: Phase 1
**Requirements**: TAG-01, TAG-02, TAG-03, CORR-01, CORR-02, CORR-03
**Success Criteria** (what must be TRUE):
  1. User can create, edit, and delete tags
  2. User can assign and remove tags from documents
  3. User can create, edit, and delete correspondents
  4. User can assign correspondent to documents
  5. User can merge duplicate correspondents into one
**Plans**: 5 plans

Plans:
- [x] 05-01-PLAN.md — Tag CRUD with management UI
- [x] 05-02-PLAN.md — Correspondent CRUD with management UI
- [x] 05-03-PLAN.md — Correspondent merge functionality
- [x] 05-04-PLAN.md — Tag assignment to documents
- [x] 05-05-PLAN.md — Correspondent assignment to documents

### Phase 6: Search
**Goal**: Users can find any document by content, tags, or correspondent
**Depends on**: Phase 3, Phase 5
**Requirements**: SEARCH-01, SEARCH-02, SEARCH-03, SEARCH-04
**Success Criteria** (what must be TRUE):
  1. User can search documents by content (full-text search)
  2. User can filter search results by tags
  3. User can filter search results by correspondent
  4. User can filter search results by date range
  5. Search results display document previews with relevant snippets
**Plans**: 3 plans

Plans:
- [x] 06-01-PLAN.md — Database search infrastructure (search_vector column, GIN index, SearchDocuments query)
- [x] 06-02-PLAN.md — Search handler and results partial
- [x] 06-03-PLAN.md — Search UI with filters and HTMX live search

### Phase 7: Network Sources
**Goal**: Documents auto-import from SMB and NFS network shares
**Depends on**: Phase 2
**Requirements**: INGEST-04, INGEST-05, ADMIN-01, ADMIN-02
**Success Criteria** (what must be TRUE):
  1. System imports PDFs from configured SMB network shares
  2. System imports PDFs from configured NFS network shares
  3. Admin can configure document sources (local, SMB, NFS paths)
  4. Admin can enable/disable individual document sources
**Plans**: 6 plans

Plans:
- [x] 07-01-PLAN.md — Database schema and credential encryption
- [x] 07-02-PLAN.md — SMB client implementation
- [x] 07-03-PLAN.md — NFS client implementation
- [x] 07-04-PLAN.md — Polling service and sync logic
- [x] 07-05-PLAN.md — Handler endpoints and UI templates
- [x] 07-06-PLAN.md — Integration wiring and navigation

### Phase 8: AI Integration
**Goal**: AI automates tagging and correspondent detection
**Depends on**: Phase 3, Phase 5
**Requirements**: AI-01, AI-02, AI-03, AI-04, QUEUE-03, QUEUE-05, ADMIN-03
**Success Criteria** (what must be TRUE):
  1. System auto-suggests tags using AI based on document content
  2. System auto-detects correspondent using AI
  3. User can configure AI provider (OpenAI, Claude, Ollama)
  4. User can configure max pages sent to AI (cost control)
  5. Dashboard shows pending/completed counts per queue
  6. User can retry failed document processing
  7. Admin can view system status and queue health
**Plans**: 6 plans

Plans:
- [x] 08-01-PLAN.md — Database schema and sqlc queries
- [x] 08-02-PLAN.md — Provider interface and implementations
- [x] 08-03-PLAN.md — AI service and job handler
- [x] 08-04-PLAN.md — Settings UI and provider configuration
- [x] 08-05-PLAN.md — Review queue and queue dashboard
- [x] 08-06-PLAN.md — Document detail integration

### Phase 9: Minimum Number of Words Import Block

**Goal**: Block document import when extracted text is below configurable word threshold
**Depends on**: Phase 3
**Requirements**: None (enhancement feature)
**Success Criteria** (what must be TRUE):

  1. Admin can configure minimum word count threshold for document import
  2. Documents with insufficient text are blocked during ingestion
  3. User is informed when document is rejected due to word count
  4. Threshold can be disabled (set to 0) for unrestricted import

**Plans**: 2 plans

Plans:
- [ ] 09-01-PLAN.md — Database migration and sqlc queries for min_word_count
- [ ] 09-02-PLAN.md — Processor validation and UI configuration

### Phase 10: Refactor to Use More templUI Components

**Goal**: Replace custom UI elements with standardized templUI components for consistency
**Depends on**: Phase 9
**Requirements**: TBD
**Success Criteria** (what must be TRUE):

  1. Custom form elements replaced with templUI components
  2. Custom modals use templUI modal component
  3. Custom buttons/inputs standardized across the app
  4. UI styling is consistent throughout the application

**Plans**: 0 plans

Plans:

- [ ] TBD (run /gsd:plan-phase 10 to break down)

### Phase 11: Dashboard

**Goal**: Real dashboard at root route with stats, counts, and links to detail pages
**Depends on**: Phase 10
**Requirements**: TBD
**Success Criteria** (what must be TRUE):

  1. Dashboard shows document count and recent uploads
  2. Dashboard shows inbox status and counts
  3. Dashboard shows queue health and pending jobs
  4. Dashboard shows tag and correspondent counts
  5. Dashboard shows AI processing stats
  6. Quick navigation links to all management pages

**Plans**: 0 plans

Plans:

- [ ] TBD (run /gsd:plan-phase 11 to break down)

### Phase 12: Queues Detail

**Goal**: Enhanced queues route with expandable details for failed jobs and recent activity
**Depends on**: Phase 11
**Requirements**: TBD
**Success Criteria** (what must be TRUE):

  1. Queues page shows all queue names with job counts
  2. Expander reveals failed jobs with error details
  3. Expander shows recent activity/completed jobs
  4. User can retry failed jobs from the detail view
  5. User can clear failed jobs from queue

**Plans**: 0 plans

Plans:

- [ ] TBD (run /gsd:plan-phase 12 to break down)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 3/3 | Complete | 2026-02-02 |
| 2. Ingestion | 5/5 | Complete | 2026-02-02 |
| 3. Processing | 5/5 | Complete | 2026-02-03 |
| 4. Viewing | 3/3 | Complete | 2026-02-03 |
| 5. Organization | 5/5 | Complete | 2026-02-03 |
| 6. Search | 3/3 | Complete | 2026-02-03 |
| 7. Network Sources | 6/6 | Complete | 2026-02-03 |
| 8. AI Integration | 6/6 | Complete | 2026-02-03 |
| 9. Minimum Word Block | 0/2 | Not Started | - |
| 10. templUI Refactor | 0/? | Not Started | - |
| 11. Dashboard | 0/? | Not Started | - |
| 12. Queues Detail | 0/? | Not Started | - |

---
*Roadmap created: 2026-02-02*
*Total v1 requirements: 35*
*All requirements mapped: yes*
