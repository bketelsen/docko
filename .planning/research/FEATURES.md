# Features Research

**Domain:** PDF Document Management System (self-hosted, household/small team)
**Researched:** 2026-02-02
**Confidence:** MEDIUM (based on training data through May 2025; web verification unavailable)

## Reference Systems Analyzed

This research draws from three mature open-source document management systems:

| System | Focus | Key Strengths |
|--------|-------|---------------|
| **Paperless-ngx** | Personal document archive | Excellent UX, auto-tagging, correspondent detection |
| **Mayan EDMS** | Enterprise document management | Workflows, granular permissions, extensive metadata |
| **Docspell** | Personal/small team | Full-text search, NLP processing, clean UI |

All three are actively maintained and represent the current state of self-hosted DMS.

---

## Table Stakes

Features users expect from any DMS. Missing these means the product feels incomplete or broken.

### Document Ingestion

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **File upload via web UI** | Basic interaction model | Low | Drag-and-drop expected |
| **Bulk upload** | Users have many documents to add initially | Low | Multi-file selection |
| **Watch folder / inbox** | Automation without manual upload | Medium | File system monitoring |
| **Duplicate detection** | Prevent clutter, wasted storage | Medium | Content hash comparison |
| **Progress indication** | Users need feedback on processing | Low | Queue status, completion % |

### Document Organization

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Tags/labels** | Universal organization paradigm | Low | Many-to-many relationship |
| **Correspondents/senders** | Who sent the document | Low | One-to-many relationship |
| **Document date** | When the document is from (not upload date) | Low | User-editable |
| **Title/name** | Human-readable identifier | Low | Auto-suggested from filename |
| **Manual editing of all metadata** | Users need to correct mistakes | Low | Edit forms for all fields |

### Search & Retrieval

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Full-text search** | Core value proposition | Medium | PostgreSQL tsvector adequate at this scale |
| **Filter by tags** | Narrow results | Low | Multi-select |
| **Filter by correspondent** | Common query pattern | Low | Dropdown or autocomplete |
| **Filter by date range** | "Documents from 2023" | Low | Date picker |
| **Search results preview** | Know what you're clicking | Medium | Snippet extraction |
| **Sort options** | Date, relevance, name | Low | Standard UI pattern |

### Document Viewing

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **In-browser PDF viewer** | No download to view | Medium | PDF.js or similar |
| **Document download** | Get the original file | Low | Direct file serving |
| **Thumbnail generation** | Visual navigation | Medium | First page rendering |
| **Multi-page preview** | Scan through document | Medium | Page thumbnails |

### Basic Processing

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Text extraction** | Enable full-text search | Medium | PDF text layer extraction |
| **Page count detection** | Basic metadata | Low | PDF library feature |
| **File size tracking** | Storage management | Low | Filesystem stat |

---

## Differentiators

Features that would make Docko stand out. Not expected, but valued when present.

### AI-Powered Organization

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Auto-tagging via AI** | Eliminate manual categorization | High | LLM integration, prompt engineering |
| **Tag suggestions** | Speed up manual workflow | Medium | Suggest, don't auto-apply |
| **Correspondent detection** | Auto-identify sender from content | High | NLP + fuzzy matching |
| **Document date extraction** | Find date within document content | Medium | Regex + heuristics + AI |
| **Document type classification** | Invoice, receipt, contract, etc. | Medium | AI or rule-based |
| **Content summarization** | Quick document overview | Medium | LLM-powered |

### Advanced Search

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Faceted search UI** | Explore document collection | Medium | Dynamic filter counts |
| **Saved searches** | Repeat common queries | Low | Store query parameters |
| **Search within results** | Iterative refinement | Low | Chain filters |
| **Boolean search operators** | Power user capability | Medium | AND/OR/NOT parsing |
| **Fuzzy matching** | Typo tolerance | Medium | PostgreSQL pg_trgm |

### Automation & Rules

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Matching rules** | Auto-tag based on content patterns | High | Rule engine, regex support |
| **Correspondent auto-assignment** | Based on learned patterns | High | ML or rule-based |
| **Scheduled imports** | Regular network share polling | Medium | Cron-like scheduling |
| **Configurable processing pipeline** | Different processing for different sources | High | Queue routing logic |

### Network Integration

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **SMB share watching** | Import from NAS without OS mounts | High | Go SMB library integration |
| **NFS share watching** | Same for NFS shares | High | Go NFS library or mount |
| **Source-specific config** | Different handling per source | Medium | Source configuration UI |

### Processing Pipeline Visibility

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Processing queue dashboard** | See what's happening | Medium | Queue status display |
| **Per-document audit trail** | Debug processing issues | Medium | Step-by-step logging |
| **Retry failed documents** | Recover from transient errors | Medium | Queue retry mechanism |
| **Processing statistics** | Understand system health | Low | Counters, averages |

### User Experience

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Keyboard navigation** | Power user efficiency | Medium | j/k navigation, shortcuts |
| **Bulk operations** | Select multiple, apply action | Medium | Multi-select UI |
| **Quick actions** | One-click common operations | Low | Action buttons |
| **Recent documents** | Quick access to recent uploads | Low | Sort by upload date |
| **Dashboard overview** | System status at a glance | Medium | Stats, recent activity |

---

## Anti-Features

Features to deliberately NOT build. Common in other DMS but wrong for this project's scope.

### Complexity Traps

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **OCR processing** | Already out of scope in PROJECT.md; PDFs have text. Adds Tesseract dependency, processing time | Defer to v2 for scanned documents |
| **Multi-tenant architecture** | Massive complexity for household use case | Single deployment, trusted users |
| **Role-based access control** | Overkill for trusted household/small team | Simple admin auth is sufficient |
| **Document versioning** | Not an editing system; archive only | Store original, immutable |
| **Check-in/check-out workflow** | Enterprise collaboration pattern | Out of scope per PROJECT.md |
| **Document editing** | This is an archive, not an editor | View and download only |
| **Complex workflow engine** | Mayan EDMS feature; enterprise focus | Simple processing pipeline |

### Infrastructure Bloat

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Elasticsearch** | Overkill at tens of thousands scale | PostgreSQL full-text search |
| **Redis queue** | Additional infrastructure | PostgreSQL-backed queue |
| **S3/cloud storage** | Out of scope per PROJECT.md | Local + network shares only |
| **Microservices** | Deployment complexity | Single binary |
| **Docker requirement** | Should run native | Docker optional, native primary |

### Scope Creep

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Mobile app** | Web responsive is sufficient for v1 | Responsive design |
| **Email ingestion** | Adds IMAP complexity | Defer; users can save PDFs manually |
| **Scanner integration** | Hardware integration complexity | Scan to watch folder instead |
| **Document templates** | Not an archive feature | Out of scope |
| **E-signature** | Different product category | Out of scope |
| **Calendar integration** | Feature creep | Out of scope |
| **Notifications/alerts** | Nice to have, not core | Defer to later |

### Premature Optimization

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Distributed processing** | Single server sufficient at this scale | Concurrent workers on one machine |
| **Caching layer** | Premature optimization | Add if needed based on profiling |
| **CDN integration** | Local network use case | Direct file serving |
| **Async search** | PostgreSQL FTS is fast enough | Sync queries initially |

---

## Feature Dependencies

Understanding which features must come first.

```
Document Ingestion (foundation)
├── File upload via web UI
├── Watch folder / inbox
│   └── Network share watching (depends on local watch working)
├── Duplicate detection
│   └── Content hash computation (prerequisite)
└── Progress indication
    └── Queue infrastructure (prerequisite)

Document Organization (requires ingestion)
├── Tags
│   ├── Manual tag assignment
│   └── Auto-tagging via AI (depends on tags existing)
├── Correspondents
│   ├── Manual correspondent assignment
│   └── Correspondent detection (depends on correspondents existing)
└── Document metadata
    └── AI-powered extraction (depends on basic metadata structure)

Search & Retrieval (requires organization)
├── Full-text search
│   └── Text extraction (prerequisite)
├── Filter by tags (depends on tags)
├── Filter by correspondent (depends on correspondents)
└── Faceted search (depends on multiple filter types)

Processing Pipeline
├── Queue infrastructure (foundation)
│   ├── Ingestion queue
│   ├── Text extraction queue (depends on ingestion)
│   ├── AI tagging queue (depends on text extraction)
│   └── Correspondent detection queue (depends on text extraction)
└── Dashboard (depends on queue infrastructure)
```

### Critical Path for MVP

1. **Queue infrastructure** - Everything depends on processing pipeline
2. **Document ingestion** - Can't do anything without documents
3. **Text extraction** - Enables search and AI features
4. **Full-text search** - Core value proposition
5. **Tags + manual assignment** - Basic organization
6. **AI tagging** - Key differentiator

---

## Complexity Assessment

Rough effort estimates for major features.

### Low Complexity (1-2 days each)

- File upload via web UI
- Bulk upload
- Tags CRUD
- Correspondents CRUD
- Manual metadata editing
- Document download
- Filter by single field
- Sort options
- Page count detection
- File size tracking
- Recent documents view
- Saved searches (store query params)
- Processing statistics (basic counters)

### Medium Complexity (3-5 days each)

- Watch folder / inbox (fsnotify)
- Duplicate detection (content hashing)
- Full-text search (PostgreSQL setup)
- Filter by date range (date picker UI)
- Search results preview (snippet extraction)
- In-browser PDF viewer (PDF.js integration)
- Thumbnail generation (imagemagick/poppler)
- Multi-page preview
- Text extraction (pdf library)
- Faceted search UI (dynamic counts)
- Boolean search operators
- Fuzzy matching (pg_trgm)
- Bulk operations UI
- Processing queue dashboard
- Per-document audit trail
- Keyboard navigation

### High Complexity (1-2 weeks each)

- SMB share watching (go-smb2 library, polling, error handling)
- NFS share watching (native or library, similar challenges)
- Auto-tagging via AI (LLM integration, prompt engineering, cost management)
- Correspondent detection (NLP, fuzzy matching, deduplication)
- Matching rules engine (rule definition, regex, action execution)
- Configurable processing pipeline (routing logic, per-source config)

---

## MVP Feature Set Recommendation

Based on PROJECT.md requirements and this analysis, the MVP should include:

### Must Have (Table Stakes)

1. Web upload with drag-and-drop
2. Watch folder for local inbox
3. Tags and correspondents (manual CRUD)
4. Full-text search
5. Basic filters (tags, correspondent, date)
6. In-browser PDF viewer
7. Document download
8. Thumbnails

### Must Have (Key Differentiators)

1. AI-powered auto-tagging
2. Processing queue with dashboard
3. Network share integration (SMB at minimum)

### Defer to Post-MVP

- OCR (per PROJECT.md)
- NFS integration (if SMB proves sufficient)
- Matching rules engine
- Correspondent auto-detection (can be manual initially)
- Saved searches
- Bulk operations
- Boolean search operators

---

## Competitive Analysis Summary

| Feature | Paperless-ngx | Mayan EDMS | Docspell | Docko Target |
|---------|---------------|------------|----------|--------------|
| Web upload | Yes | Yes | Yes | Yes |
| Watch folder | Yes | No (API) | Yes | Yes |
| Full-text search | Yes | Yes | Yes | Yes |
| Tags | Yes | Yes | Yes | Yes |
| Correspondents | Yes | No (metadata) | Person/Org | Yes |
| Auto-tagging | Rules + ML | Workflow | NLP | AI (LLM) |
| OCR | Yes (Tesseract) | Yes | Yes | No (v1) |
| Network shares | Consume folder | No | No | Yes (differentiator) |
| Processing queue | Yes | Yes | Yes | Yes |
| RBAC | No | Yes | No | No |
| Workflows | No | Yes | No | No |

**Docko's Differentiation:** Native network share integration (SMB/NFS) without OS mounts, and modern LLM-powered tagging instead of traditional ML/rules. Simpler than Mayan, more network-aware than Paperless-ngx/Docspell.

---

## Sources and Confidence

| Claim | Confidence | Basis |
|-------|------------|-------|
| Paperless-ngx features | MEDIUM | Training data (May 2025); project was mature |
| Mayan EDMS features | MEDIUM | Training data; enterprise features well-documented |
| Docspell features | MEDIUM | Training data; less well-known but analyzed |
| Complexity estimates | LOW-MEDIUM | Based on general software development experience |
| Market expectations | MEDIUM | Based on DMS ecosystem patterns |

**Validation recommended:** Before implementation, verify current feature sets of reference systems via their official documentation, as features may have been added or changed since May 2025.

---

*Research date: 2026-02-02*
*Note: Web verification tools unavailable during research. Findings based on training data through May 2025.*
