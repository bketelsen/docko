# Requirements Archive: v1.0 Initial Release

**Archived:** 2026-02-04
**Status:** SHIPPED

This is the archived requirements specification for v1.0.
For current requirements, see `.planning/REQUIREMENTS.md` (created for next milestone).

---

## v1 Requirements

Requirements for initial release. All mapped to roadmap phases and completed.

### Document Ingestion (7/7)

- [x] **INGEST-01**: User can upload PDF files via web UI with drag-and-drop
- [x] **INGEST-02**: User can upload multiple files at once (bulk upload)
- [x] **INGEST-03**: System watches local inbox directory for new PDFs
- [x] **INGEST-04**: System imports PDFs from configured SMB network shares
- [x] **INGEST-05**: System imports PDFs from configured NFS network shares
- [x] **INGEST-06**: System detects duplicate documents by content hash (SHA-256)
- [x] **INGEST-07**: User can configure duplicate handling per source (delete, rename, skip)

### Document Store (3/3)

- [x] **STORE-01**: Documents are assigned UUID and stored in organized directory structure
- [x] **STORE-02**: Original files are preserved unmodified in originals/ directory
- [x] **STORE-03**: Document metadata is stored in database (filename, size, page count)

### Tags & Correspondents (6/6)

- [x] **TAG-01**: User can create, edit, and delete tags
- [x] **TAG-02**: User can assign tags to documents manually
- [x] **TAG-03**: User can remove tags from documents
- [x] **CORR-01**: User can create, edit, and delete correspondents
- [x] **CORR-02**: User can assign correspondent to document manually
- [x] **CORR-03**: User can merge duplicate correspondents

### Search & Retrieval (4/4)

- [x] **SEARCH-01**: User can search documents by content (full-text search)
- [x] **SEARCH-02**: User can filter search results by tags
- [x] **SEARCH-03**: User can filter search results by correspondent
- [x] **SEARCH-04**: User can filter search results by date range

### Document Viewing (3/3)

- [x] **VIEW-01**: User can view PDF in browser without downloading
- [x] **VIEW-02**: User can download original PDF file
- [x] **VIEW-03**: Documents display thumbnail preview (first page)

### Processing Pipeline (5/5)

- [x] **QUEUE-01**: Document processing uses queue-based architecture
- [x] **QUEUE-02**: Text is extracted from PDFs and indexed for search
- [x] **QUEUE-03**: Dashboard shows pending/completed counts per queue
- [x] **QUEUE-04**: Each document has audit trail of processing steps
- [x] **QUEUE-05**: User can retry failed document processing

### AI Features (4/4)

- [x] **AI-01**: System auto-suggests tags using AI (LLM integration)
- [x] **AI-02**: System auto-detects correspondent using AI
- [x] **AI-03**: User can configure AI provider (OpenAI, Claude, Ollama)
- [x] **AI-04**: User can configure max pages sent to AI (cost control)

### Admin & Configuration (3/3)

- [x] **ADMIN-01**: Admin can configure document sources (local, SMB, NFS)
- [x] **ADMIN-02**: Admin can enable/disable document sources
- [x] **ADMIN-03**: Admin can view system status and queue health

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INGEST-01 | Phase 2 | Complete |
| INGEST-02 | Phase 2 | Complete |
| INGEST-03 | Phase 2 | Complete |
| INGEST-04 | Phase 7 | Complete |
| INGEST-05 | Phase 7 | Complete |
| INGEST-06 | Phase 2 | Complete |
| INGEST-07 | Phase 2 | Complete |
| STORE-01 | Phase 1 | Complete |
| STORE-02 | Phase 1 | Complete |
| STORE-03 | Phase 1 | Complete |
| TAG-01 | Phase 5 | Complete |
| TAG-02 | Phase 5 | Complete |
| TAG-03 | Phase 5 | Complete |
| CORR-01 | Phase 5 | Complete |
| CORR-02 | Phase 5 | Complete |
| CORR-03 | Phase 5 | Complete |
| SEARCH-01 | Phase 6 | Complete |
| SEARCH-02 | Phase 6 | Complete |
| SEARCH-03 | Phase 6 | Complete |
| SEARCH-04 | Phase 6 | Complete |
| VIEW-01 | Phase 4 | Complete |
| VIEW-02 | Phase 4 | Complete |
| VIEW-03 | Phase 3 | Complete |
| QUEUE-01 | Phase 1 | Complete |
| QUEUE-02 | Phase 3 | Complete |
| QUEUE-03 | Phase 8 | Complete |
| QUEUE-04 | Phase 1 | Complete |
| QUEUE-05 | Phase 8 | Complete |
| AI-01 | Phase 8 | Complete |
| AI-02 | Phase 8 | Complete |
| AI-03 | Phase 8 | Complete |
| AI-04 | Phase 8 | Complete |
| ADMIN-01 | Phase 7 | Complete |
| ADMIN-02 | Phase 7 | Complete |
| ADMIN-03 | Phase 8 | Complete |

## Milestone Summary

**Shipped:** 35 of 35 v1 requirements
**Adjusted:** None — all requirements implemented as specified
**Dropped:** None

## Out of Scope (preserved for reference)

| Feature | Reason |
|---------|--------|
| Multi-tenant architecture | Massive complexity for household use case |
| Role-based access control | Overkill for trusted household/small team |
| Document versioning | Not an editing system; archive only |
| Document editing | This is an archive, not an editor |
| Complex workflow engine | Enterprise feature, not needed |
| Elasticsearch | PostgreSQL FTS sufficient at this scale |
| Redis queue | PostgreSQL-backed queue simpler |
| S3/cloud storage | Local + network shares only for v1 |
| Mobile app | Web responsive is sufficient |
| Email ingestion | Users can save PDFs manually |
| Scanner integration | Scan to watch folder instead |
| E-signature | Different product category |
| Real-time notifications | Nice to have, not core |

---

*Archived: 2026-02-04 as part of v1.0 milestone completion*
