---
phase: 03-processing
plan: 02
subsystem: processing
tags: [pdf, text-extraction, ocr, ocrmypdf, ledongthuc-pdf]

# Dependency graph
requires:
  - phase: 03-01
    provides: OCRmyPDF Docker service infrastructure
provides:
  - TextExtractor with embedded text extraction using ledongthuc/pdf
  - OCR fallback via OCRmyPDF service through shared volumes
  - Text extraction returns content and method (embedded/ocr)
affects: [03-04, 03-05, 06-search]

# Tech tracking
tech-stack:
  added: [github.com/ledongthuc/pdf]
  patterns: [shared-volume-ipc, polling-with-timeout, extraction-fallback]

key-files:
  created:
    - internal/processing/text.go
    - internal/processing/text_test.go
  modified:
    - docker-compose.yml (bind mounts for OCR volumes)
    - go.mod
    - go.sum

key-decisions:
  - "Use bind mounts instead of Docker named volumes for OCR communication"
  - "100-char threshold to determine if embedded text is sufficient"
  - "5-minute timeout for OCR processing"
  - "Poll every 500ms for OCR completion"

patterns-established:
  - "Shared volume IPC: App writes to storage/ocr-input, OCRmyPDF watches and writes to storage/ocr-output"
  - "Extraction fallback: Try fast embedded extraction first, fall back to slower OCR only when needed"

# Metrics
duration: 3min
completed: 2026-02-03
---

# Phase 3 Plan 2: Text Extraction Summary

**Text extraction service using ledongthuc/pdf for embedded text with OCRmyPDF service fallback via shared volumes**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-03T01:54:33Z
- **Completed:** 2026-02-03T01:57:33Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- TextExtractor extracts embedded text from PDFs using ledongthuc/pdf library
- OCR fallback communicates with OCRmyPDF Docker service via shared volumes (no docker run)
- Extract() returns both text content and method used ("embedded" or "ocr")
- Comprehensive test suite covering extraction, timeout, context cancellation, and cleanup

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement embedded text extraction** - `474992a` (feat)
2. **Task 2: Implement OCR fallback via OCRmyPDF service** - `53a4bc3` (test)

## Files Created/Modified

- `internal/processing/text.go` - TextExtractor with embedded extraction and OCR fallback
- `internal/processing/text_test.go` - Comprehensive tests for text extraction
- `docker-compose.yml` - Changed OCR volumes from named to bind mounts
- `go.mod` - Added ledongthuc/pdf dependency
- `go.sum` - Updated checksums

## Decisions Made

1. **Bind mounts instead of Docker named volumes** - Named volumes require root access from host. Bind mounts to storage/ocr-input and storage/ocr-output allow the Go app to communicate with the OCRmyPDF service without elevated privileges.

2. **100-character minimum text threshold** - PDFs with less than 100 characters of extracted embedded text are considered to need OCR. This threshold avoids treating PDFs with just metadata or whitespace as "having text."

3. **5-minute OCR timeout** - Per RESEARCH.md recommendation, OCR operations can take several minutes for large scanned documents. 5 minutes provides reasonable headroom.

4. **500ms polling interval** - Balance between responsiveness (quick detection of completed OCR) and avoiding excessive disk I/O from stat() calls.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Changed Docker volumes to bind mounts**
- **Found during:** Task 1 (infrastructure setup)
- **Issue:** Docker named volumes at /var/lib/docker/volumes/* require root access, preventing the Go app from reading/writing OCR files
- **Fix:** Modified docker-compose.yml to use bind mounts (./storage/ocr-input:/input and ./storage/ocr-output:/output)
- **Files modified:** docker-compose.yml
- **Verification:** Created directories with proper permissions, OCRmyPDF service starts successfully with bind mounts
- **Committed in:** 474992a (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Blocking fix required for app-service communication. No scope creep.

## Issues Encountered

None - plan executed smoothly after fixing the volume mount approach.

## User Setup Required

None - no external service configuration required. OCRmyPDF service is configured via docker-compose.yml.

## Next Phase Readiness

- TextExtractor ready for use by processing job handler (03-04)
- OCRmyPDF service operational and tested
- Next: Plan 03-03 (thumbnails) already complete, 03-04 (processing job handler) can begin

---
*Phase: 03-processing*
*Completed: 2026-02-03*
