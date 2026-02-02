# Roadmap: Docko

## Overview

Docko delivers a PDF document management system in 8 phases, progressing from foundation through ingestion, processing, organization, search, and finally AI-powered automation. The journey starts with reliable document storage and queue infrastructure, then builds ingestion pipelines, text extraction, and viewing capabilities. Organization (tags/correspondents) and search follow, enabling the core value of "find any document instantly." Network shares and AI tagging complete the system, delivering the automation that eliminates manual filing.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Foundation** - Document store structure and queue infrastructure
- [ ] **Phase 2: Ingestion** - Web upload and local inbox watching
- [ ] **Phase 3: Processing** - Text extraction and thumbnail generation
- [ ] **Phase 4: Viewing** - PDF preview and download
- [ ] **Phase 5: Organization** - Tags and correspondents management
- [ ] **Phase 6: Search** - Full-text search with filtering
- [ ] **Phase 7: Network Sources** - SMB and NFS share integration
- [ ] **Phase 8: AI Integration** - Auto-tagging and correspondent detection

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
- [ ] 01-01-PLAN.md — Database schema and storage service
- [ ] 01-02-PLAN.md — Job queue implementation
- [ ] 01-03-PLAN.md — Document service and integration

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
**Plans**: TBD

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD

### Phase 3: Processing
**Goal**: Uploaded documents are processed for text content and thumbnails
**Depends on**: Phase 2
**Requirements**: QUEUE-02, VIEW-03
**Success Criteria** (what must be TRUE):
  1. Text is extracted from PDFs and indexed in database for search
  2. Thumbnail (first page preview) is generated for each document
  3. Processing happens asynchronously via queue (does not block upload)
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

### Phase 4: Viewing
**Goal**: Users can view and download documents
**Depends on**: Phase 3
**Requirements**: VIEW-01, VIEW-02
**Success Criteria** (what must be TRUE):
  1. User can view PDF in browser without downloading
  2. User can download the original PDF file
  3. Document detail page shows metadata and thumbnail
**Plans**: TBD

Plans:
- [ ] 04-01: TBD

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
**Plans**: TBD

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

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
**Plans**: TBD

Plans:
- [ ] 06-01: TBD
- [ ] 06-02: TBD

### Phase 7: Network Sources
**Goal**: Documents auto-import from SMB and NFS network shares
**Depends on**: Phase 2
**Requirements**: INGEST-04, INGEST-05, ADMIN-01, ADMIN-02
**Success Criteria** (what must be TRUE):
  1. System imports PDFs from configured SMB network shares
  2. System imports PDFs from configured NFS network shares
  3. Admin can configure document sources (local, SMB, NFS paths)
  4. Admin can enable/disable individual document sources
**Plans**: TBD

Plans:
- [ ] 07-01: TBD
- [ ] 07-02: TBD

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
**Plans**: TBD

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD
- [ ] 08-03: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 0/3 | Planned | - |
| 2. Ingestion | 0/2 | Not started | - |
| 3. Processing | 0/2 | Not started | - |
| 4. Viewing | 0/1 | Not started | - |
| 5. Organization | 0/2 | Not started | - |
| 6. Search | 0/2 | Not started | - |
| 7. Network Sources | 0/2 | Not started | - |
| 8. AI Integration | 0/3 | Not started | - |

---
*Roadmap created: 2026-02-02*
*Total v1 requirements: 35*
*All requirements mapped: yes*
