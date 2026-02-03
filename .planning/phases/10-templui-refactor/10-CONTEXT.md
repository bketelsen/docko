# Phase 10: Refactor to Use More templUI Components - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace custom UI elements with standardized templUI components for consistency across the entire application. This includes forms, buttons, interactive elements, static display components, navigation, tables, tabs, and toasts. The goal is visual and behavioral consistency, not new functionality.

</domain>

<decisions>
## Implementation Decisions

### Component Scope
- Replace ALL custom elements at once (not incremental by page)
- Full standardization: forms, buttons, dropdowns, and all interactive elements
- Include static display components: cards, badges, alerts
- Include navigation and sidebar structure
- Use templUI table component for document lists and search results
- Use templUI tabs component for document detail page
- Use templUI toast component for notifications

### Migration Approach
- Adapt our patterns to match templUI component behavior (even if slightly different from current)

### Custom vs templUI Balance
- When templUI lacks a needed component: build new component following templUI patterns/styling
- Custom components go in separate directory (`components/custom/`)
- Custom components must follow templUI naming conventions and prop patterns

### Visual Consistency
- Adopt templUI's default color scheme
- Use strict templUI spacing system everywhere (accept layout shifts)
- Use templUI's hover, focus, and active states everywhere
- Dark mode support must work correctly for all components

### Claude's Discretion
- Loading states and skeletons (evaluate if templUI has good options)
- Migration ordering (by component type vs by page)
- Commit boundaries (atomic per component vs batch by page)
- Cleanup of unused custom component files
- PDF viewer modal refactor (evaluate if effort is worthwhile)

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-templui-refactor*
*Context gathered: 2026-02-03*
