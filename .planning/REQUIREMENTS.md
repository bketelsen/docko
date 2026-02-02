# Requirements: Docko

**Defined:** 2026-02-02
**Core Value:** Find any document instantly AND automate the tagging/filing that's currently manual

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Document Ingestion

- [ ] **INGEST-01**: User can upload PDF files via web UI with drag-and-drop
- [ ] **INGEST-02**: User can upload multiple files at once (bulk upload)
- [ ] **INGEST-03**: System watches local inbox directory for new PDFs
- [ ] **INGEST-04**: System imports PDFs from configured SMB network shares
- [ ] **INGEST-05**: System imports PDFs from configured NFS network shares
- [ ] **INGEST-06**: System detects duplicate documents by content hash (SHA-256)
- [ ] **INGEST-07**: User can configure duplicate handling per source (delete, rename, skip)

### Document Store

- [ ] **STORE-01**: Documents are assigned UUID and stored in organized directory structure
- [ ] **STORE-02**: Original files are preserved unmodified in originals/ directory
- [ ] **STORE-03**: Document metadata is stored in database (filename, size, page count)

### Tags & Correspondents

- [ ] **TAG-01**: User can create, edit, and delete tags
- [ ] **TAG-02**: User can assign tags to documents manually
- [ ] **TAG-03**: User can remove tags from documents
- [ ] **CORR-01**: User can create, edit, and delete correspondents
- [ ] **CORR-02**: User can assign correspondent to document manually
- [ ] **CORR-03**: User can merge duplicate correspondents

### Search & Retrieval

- [ ] **SEARCH-01**: User can search documents by content (full-text search)
- [ ] **SEARCH-02**: User can filter search results by tags
- [ ] **SEARCH-03**: User can filter search results by correspondent
- [ ] **SEARCH-04**: User can filter search results by date range

### Document Viewing

- [ ] **VIEW-01**: User can view PDF in browser without downloading
- [ ] **VIEW-02**: User can download original PDF file
- [ ] **VIEW-03**: Documents display thumbnail preview (first page)

### Processing Pipeline

- [ ] **QUEUE-01**: Document processing uses queue-based architecture
- [ ] **QUEUE-02**: Text is extracted from PDFs and indexed for search
- [ ] **QUEUE-03**: Dashboard shows pending/completed counts per queue
- [ ] **QUEUE-04**: Each document has audit trail of processing steps
- [ ] **QUEUE-05**: User can retry failed document processing

### AI Features

- [ ] **AI-01**: System auto-suggests tags using AI (LLM integration)
- [ ] **AI-02**: System auto-detects correspondent using AI
- [ ] **AI-03**: User can configure AI provider (OpenAI, Claude, Ollama)
- [ ] **AI-04**: User can configure max pages sent to AI (cost control)

### Admin & Configuration

- [ ] **ADMIN-01**: Admin can configure document sources (local, SMB, NFS)
- [ ] **ADMIN-02**: Admin can enable/disable document sources
- [ ] **ADMIN-03**: Admin can view system status and queue health

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Search

- **SEARCH-05**: Faceted search UI with dynamic filter counts
- **SEARCH-06**: Saved searches (store and reuse query parameters)
- **SEARCH-07**: Fuzzy matching for typo tolerance (pg_trgm)
- **SEARCH-08**: Boolean search operators (AND/OR/NOT)

### Advanced Viewing

- **VIEW-04**: Multi-page preview (thumbnails of all pages)

### Advanced Organization

- **TAG-04**: Matching rules engine (auto-tag based on content patterns)
- **CORR-04**: Correspondent alias management

### Processing

- **QUEUE-06**: Configurable concurrency per queue type

### OCR

- **OCR-01**: OCR for scanned documents (Tesseract integration)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

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

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INGEST-01 | TBD | Pending |
| INGEST-02 | TBD | Pending |
| INGEST-03 | TBD | Pending |
| INGEST-04 | TBD | Pending |
| INGEST-05 | TBD | Pending |
| INGEST-06 | TBD | Pending |
| INGEST-07 | TBD | Pending |
| STORE-01 | TBD | Pending |
| STORE-02 | TBD | Pending |
| STORE-03 | TBD | Pending |
| TAG-01 | TBD | Pending |
| TAG-02 | TBD | Pending |
| TAG-03 | TBD | Pending |
| CORR-01 | TBD | Pending |
| CORR-02 | TBD | Pending |
| CORR-03 | TBD | Pending |
| SEARCH-01 | TBD | Pending |
| SEARCH-02 | TBD | Pending |
| SEARCH-03 | TBD | Pending |
| SEARCH-04 | TBD | Pending |
| VIEW-01 | TBD | Pending |
| VIEW-02 | TBD | Pending |
| VIEW-03 | TBD | Pending |
| QUEUE-01 | TBD | Pending |
| QUEUE-02 | TBD | Pending |
| QUEUE-03 | TBD | Pending |
| QUEUE-04 | TBD | Pending |
| QUEUE-05 | TBD | Pending |
| AI-01 | TBD | Pending |
| AI-02 | TBD | Pending |
| AI-03 | TBD | Pending |
| AI-04 | TBD | Pending |
| ADMIN-01 | TBD | Pending |
| ADMIN-02 | TBD | Pending |
| ADMIN-03 | TBD | Pending |

**Coverage:**
- v1 requirements: 35 total
- Mapped to phases: 0 (pending roadmap creation)
- Unmapped: 35

---
*Requirements defined: 2026-02-02*
*Last updated: 2026-02-02 after initial definition*
