# Phase 4: Viewing - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can view PDFs in-browser and download original files. Document detail page displays metadata and thumbnail. Creating, editing, organizing, or searching documents are separate phases.

</domain>

<decisions>
## Implementation Decisions

### PDF viewer experience
- Modal/overlay viewer (not embedded or full-page)
- Standard controls: zoom in/out, page navigation, close, fullscreen option
- Responsive on mobile — same modal adapts to smaller screens
- Multiple dismiss methods: close button, Escape key, click outside backdrop

### Detail page layout
- Side-by-side layout: thumbnail on one side, metadata panel on the other
- Show all available metadata
- Organized in tabs:
  - **Overview tab** (default): user-facing info — name, page count, date added, processing status
  - **Technical tab**: system info — content hash, storage paths, OCR status/details

### Action UX
- Buttons appear in both locations: primary toolbar at top, secondary/icon buttons on thumbnail
- View is primary action (filled button), Download is secondary (outline button)
- Download uses original filename exactly as uploaded
- Clicking thumbnail navigates to detail page (does not open viewer directly)

### Navigation flow
- Entry points: document list page + "recently added" section on dashboard
- Breadcrumb navigation: Home > Documents > [Document Name]
- URL structure: `/documents/:id` (simple UUID-based)

### Claude's Discretion
- Next/previous document navigation on detail page (arrows or return to list)
- Exact spacing and responsive breakpoints
- Loading states and error handling
- Keyboard shortcuts beyond Escape for modal

</decisions>

<specifics>
## Specific Ideas

No specific references — open to standard document viewer patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-viewing*
*Context gathered: 2026-02-02*
