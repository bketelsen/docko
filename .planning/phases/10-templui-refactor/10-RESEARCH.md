# Phase 10: Refactor to Use More templUI Components - Research

**Researched:** 2026-02-03
**Domain:** templUI component library migration for Go/templ application
**Confidence:** HIGH

## Summary

This research investigates migrating custom UI elements to the templUI component library (v1.4.0) for visual and behavioral consistency across a Go/Echo/templ/HTMX application. The project already uses templUI for some components (button, input, label, card, tabs, dialog, toast, breadcrumb, aspectratio) but has significant custom UI code that should be replaced.

templUI provides 41+ components with consistent styling, dark mode support, and JavaScript integration via per-component scripts. The project's existing CSS theme variables and Tailwind configuration are already aligned with templUI's design system (using oklch colors, CSS custom properties for --background, --foreground, --primary, etc.).

**Primary recommendation:** Install missing templUI components (table, sidebar, badge, alert, skeleton, selectbox, switch, checkbox, textarea, form, dropdown) and systematically replace all custom UI elements, starting with form components and navigation, then tables, then notifications.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| templUI | v1.4.0 | UI component library | Already in use, provides 41+ consistent components |
| templ | current | Go HTML templating | Project standard |
| Tailwind CSS | v4 | Utility-first CSS | Already configured with templUI theme |
| tailwind-merge-go | latest | Class conflict resolution | Required by templUI utils |

### Currently Installed templUI Components
| Component | Status | Notes |
|-----------|--------|-------|
| button | Installed | Used in login page |
| input | Installed | Used in login page |
| label | Installed | Used in login page |
| card | Installed | Used in login page |
| tabs | Installed | Used in document detail page |
| dialog | Installed | Available but custom modals also exist |
| toast | Installed | Available but custom toast container exists |
| breadcrumb | Installed | Used in document detail page |
| aspectratio | Installed | Available |
| icon | Installed | Used by other components |

### Components to Install
| Component | Purpose | Replaces |
|-----------|---------|----------|
| table | Document lists, search results | Custom `<table>` markup |
| sidebar | Admin navigation | Custom `AdminSidebar()` template |
| badge | Status indicators, tag pills | Custom status badges, tag chips |
| alert | Error/success messages | Custom error divs |
| skeleton | Loading placeholders | Custom loading spinners |
| selectbox | Dropdown selects | Custom `<select>` elements |
| switch | Toggle controls | Custom toggle buttons (inboxes) |
| checkbox | Form checkboxes | Custom checkbox markup |
| textarea | Multi-line inputs | Custom `<textarea>` elements |
| form | Form containers | Custom form markup |
| dropdown | Action menus | Custom dropdown menus |
| sheet | Slide-out panels | Custom modals (PDF viewer) |
| pagination | Table pagination | Custom pagination buttons |
| separator | Visual dividers | Custom `<hr>` and border elements |
| tooltip | Hover hints | Custom `title` attributes |
| popover | Floating content | N/A (dependency of selectbox) |

**Installation:**
```bash
templui add table sidebar badge alert skeleton selectbox switch checkbox textarea form dropdown sheet pagination separator tooltip
```

## Architecture Patterns

### Recommended Component Organization
```
components/                # templUI components (auto-managed)
  button/
  input/
  table/
  sidebar/
  ...
components/custom/         # Project-specific components following templUI patterns
  pdf-viewer/
  status-badge/           # If templUI badge doesn't fit exactly
  ...
utils/
  templui.go              # TwMerge, If, IfElse, RandomID utilities
assets/js/                # templUI JavaScript (auto-managed)
  tabs.min.js
  dialog.min.js
  ...
```

### Pattern 1: Props-Based Component Usage
**What:** Pass configuration via Props struct, use children for content
**When to use:** All templUI component invocations
**Example:**
```go
// Source: Existing project pattern (login.templ)
@button.Button(button.Props{
    Type:      button.TypeSubmit,
    Variant:   button.VariantDefault,
    FullWidth: true,
}) {
    Sign In
}

@input.Input(input.Props{
    ID:          "username",
    Type:        input.TypeText,
    Name:        "username",
    Placeholder: "admin",
    Attributes:  templ.Attributes{"required": "true"},
})
```

### Pattern 2: HTMX Attributes via templ.Attributes
**What:** Pass HTMX attributes through the Attributes prop
**When to use:** Any component that needs HTMX behavior
**Example:**
```go
// Correct: HTMX attributes via Attributes prop
@button.Button(button.Props{
    Variant: button.VariantDestructive,
    Attributes: templ.Attributes{
        "hx-delete": fmt.Sprintf("/tags/%s", tag.ID.String()),
        "hx-target": fmt.Sprintf("#tag-%s", tag.ID.String()),
        "hx-swap":   "outerHTML",
        "hx-confirm": "Are you sure?",
    },
}) {
    Delete
}
```

### Pattern 3: Table Component Structure
**What:** Use templUI table with semantic sub-components
**When to use:** Document lists, search results
**Example:**
```go
// Source: templUI table component structure
@table.Table() {
    @table.Header() {
        @table.Row() {
            @table.Head() { Document }
            @table.Head() { Tags }
            @table.Head() { Status }
        }
    }
    @table.Body() {
        for _, doc := range documents {
            @table.Row() {
                @table.Cell() { doc.Filename }
                @table.Cell() { @tagBadges(doc.Tags) }
                @table.Cell() { @statusBadge(doc.Status) }
            }
        }
    }
}
```

### Pattern 4: Sidebar Component Structure
**What:** Use templUI sidebar with layout wrapper
**When to use:** Admin layout navigation
**Example:**
```go
// Source: templUI sidebar component structure
@sidebar.Layout() {
    @sidebar.Sidebar(sidebar.Props{
        Side:        sidebar.SideLeft,
        Variant:     sidebar.VariantSidebar,
        Collapsible: sidebar.CollapsibleIcon,
    }) {
        @sidebar.Header() {
            Site Name
        }
        @sidebar.Content() {
            @sidebar.Group() {
                @sidebar.Menu() {
                    @sidebar.MenuItem() {
                        @sidebar.MenuButton(sidebar.MenuButtonProps{Href: "/"}) {
                            @icon.Home() Dashboard
                        }
                    }
                }
            }
        }
    }
    // Main content area
    <main>{ children... }</main>
}
```

### Pattern 5: Badge for Status Indicators
**What:** Use templUI badge with appropriate variants
**When to use:** Processing status, tag display, counts
**Example:**
```go
// Source: templUI badge component
@badge.Badge(badge.Props{Variant: badge.VariantDefault}) {
    Completed
}

@badge.Badge(badge.Props{Variant: badge.VariantDestructive}) {
    Failed
}

@badge.Badge(badge.Props{Variant: badge.VariantSecondary}) {
    3 documents
}
```

### Pattern 6: Switch for Toggle Controls
**What:** Use templUI switch component for on/off toggles
**When to use:** Inbox enable/disable, settings toggles
**Example:**
```go
// Source: templUI switch component
@switch.Switch(switch.Props{
    Name:    "enabled",
    Checked: inbox.Enabled,
    Attributes: templ.Attributes{
        "hx-post":   fmt.Sprintf("/inboxes/%s/toggle", inbox.ID),
        "hx-target": fmt.Sprintf("#inbox-%s", inbox.ID),
        "hx-swap":   "outerHTML",
    },
})
```

### Pattern 7: SelectBox for Dropdowns
**What:** Use templUI selectbox for dropdown selections
**When to use:** Filter dropdowns, correspondent select, date range
**Example:**
```go
// Source: templUI selectbox component
@selectbox.SelectBox() {
    @selectbox.Trigger(selectbox.TriggerProps{
        Name: "correspondent",
    }) {
        @selectbox.Value() {
            All correspondents
        }
    }
    @selectbox.Content() {
        @selectbox.Item(selectbox.ItemProps{Value: ""}) {
            All correspondents
        }
        for _, corr := range correspondents {
            @selectbox.Item(selectbox.ItemProps{
                Value:    corr.ID.String(),
                Selected: params.CorrespondentID == corr.ID.String(),
            }) {
                { corr.Name }
            }
        }
    }
}
```

### Anti-Patterns to Avoid
- **Mixing custom and templUI styling:** Don't apply custom Tailwind classes that override templUI's design tokens
- **Using native `<select>` when templUI selectbox fits:** Native selects have limited styling options
- **Building custom toggle switches:** Use templUI switch component
- **Hand-rolling status badges:** Use templUI badge with variants
- **Custom modal implementations:** Use templUI dialog with HTMX triggers

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Status badges | Custom `<span>` with colors | `badge.Badge` | Consistent variants, dark mode support |
| Toggle switches | Custom button with translate | `switch.Switch` | Accessibility, keyboard support |
| Dropdown menus | Custom divs with JS | `dropdown.Dropdown` or `selectbox.SelectBox` | Positioning, keyboard navigation, click-away |
| Data tables | Raw `<table>` HTML | `table.Table` | Consistent styling, selection support |
| Loading states | Custom spinners | `skeleton.Skeleton` | Consistent animation, sizing |
| Error messages | Custom red divs | `alert.Alert` | Variants, icons, accessibility |
| Navigation sidebar | Custom `<aside>` | `sidebar.Sidebar` | Collapse behavior, mobile support |
| Confirmation dialogs | JS `confirm()` | `dialog.Dialog` | Customizable, consistent styling |

**Key insight:** templUI components handle dark mode, focus states, disabled states, and accessibility automatically. Custom implementations frequently miss edge cases (keyboard navigation, screen readers, high contrast mode).

## Common Pitfalls

### Pitfall 1: Forgetting Script() Includes
**What goes wrong:** Interactive components (tabs, dialog, selectbox) don't work
**Why it happens:** templUI uses per-component JavaScript, must be included
**How to avoid:** Always add `@componentName.Script()` in template or layout
**Warning signs:** Click handlers not firing, dropdowns not opening

### Pitfall 2: HTMX Attribute Casing
**What goes wrong:** HTMX attributes don't trigger
**Why it happens:** Using wrong case in templ.Attributes
**How to avoid:** Use lowercase: `"hx-post"`, `"hx-target"`, `"hx-swap"`
**Warning signs:** No network requests on interaction

### Pitfall 3: Missing Form Integration
**What goes wrong:** Form values not submitted
**Why it happens:** templUI input components need proper name attributes
**How to avoid:** Always set `Name` prop on form inputs
**Warning signs:** Empty request bodies

### Pitfall 4: Dark Mode Color Overrides
**What goes wrong:** Custom colors don't adapt to dark mode
**Why it happens:** Using hard-coded colors instead of CSS variables
**How to avoid:** Use Tailwind classes that reference theme variables: `text-foreground`, `bg-background`, `border-border`
**Warning signs:** Unreadable text in dark mode

### Pitfall 5: Dialog Not Closing After HTMX
**What goes wrong:** Dialog stays open after form submission
**Why it happens:** HTMX swap doesn't trigger dialog close
**How to avoid:** Use HX-Trigger header to emit close event, or use `hx-on::after-request`
**Warning signs:** Users manually closing dialogs

### Pitfall 6: SelectBox HTMX Trigger
**What goes wrong:** SelectBox changes don't trigger HTMX requests
**Why it happens:** SelectBox is not a native `<select>`, events differ
**How to avoid:** Use `hx-trigger="change"` on the SelectBox, or use custom events
**Warning signs:** Filter changes don't update results

## Code Examples

Verified patterns from the existing codebase and templUI documentation:

### Login Form with templUI Components
```go
// Source: templates/pages/admin/login.templ (existing)
@card.Card() {
    @card.Header() {
        @card.Title() { Admin Login }
        @card.Description() { Enter your credentials. }
    }
    @card.Content() {
        <form method="POST" action="/login" class="space-y-4">
            <div class="space-y-2">
                @label.Label(label.Props{For: "username"}) { Username }
                @input.Input(input.Props{
                    ID:          "username",
                    Type:        input.TypeText,
                    Name:        "username",
                    Placeholder: "admin",
                    Attributes:  templ.Attributes{"required": "true"},
                })
            </div>
            @button.Button(button.Props{
                Type:      button.TypeSubmit,
                FullWidth: true,
            }) { Sign In }
        </form>
    }
}
@input.Script()
```

### Document Detail Tabs
```go
// Source: templates/pages/admin/document_detail.templ (existing)
@tabs.Tabs(tabs.Props{ID: "doc-details"}) {
    @tabs.List() {
        @tabs.Trigger(tabs.TriggerProps{Value: "overview", IsActive: true}) {
            Overview
        }
        @tabs.Trigger(tabs.TriggerProps{Value: "technical"}) {
            Technical
        }
    }
    @tabs.Content(tabs.ContentProps{Value: "overview", IsActive: true}) {
        // Overview content
    }
    @tabs.Content(tabs.ContentProps{Value: "technical"}) {
        // Technical content
    }
}
@tabs.Script()
```

### Button with HTMX Delete
```go
// Pattern for HTMX-enabled buttons
@button.Button(button.Props{
    Variant: button.VariantDestructive,
    Size:    button.SizeSm,
    Attributes: templ.Attributes{
        "hx-delete":  fmt.Sprintf("/items/%s", item.ID),
        "hx-target":  fmt.Sprintf("#item-%s", item.ID),
        "hx-swap":    "outerHTML",
        "hx-confirm": "Delete this item?",
    },
}) {
    @icon.Trash(icon.Props{Size: 16})
    Delete
}
```

### Alert Component Usage
```go
// Pattern for error/success messages
@alert.Alert(alert.Props{Variant: alert.VariantDestructive}) {
    @alert.Title() { Error }
    @alert.Description() { { errorMsg } }
}

// Or inline with icon
@alert.Alert() {
    @icon.CheckCircle(icon.Props{Size: 16, Class: "text-green-500"})
    @alert.Description() { Settings saved successfully. }
}
```

### Table with HTMX Row Updates
```go
// Pattern for document list table
@table.Table(table.Props{Class: "w-full"}) {
    @table.Header() {
        @table.Row() {
            @table.Head() { Document }
            @table.Head() { Status }
            @table.Head() { Actions }
        }
    }
    @table.Body() {
        for _, doc := range documents {
            @table.Row(table.RowProps{
                Attributes: templ.Attributes{
                    "id": fmt.Sprintf("doc-%s", doc.ID),
                },
            }) {
                @table.Cell() { doc.Filename }
                @table.Cell() {
                    <div sse-swap={ "doc-" + doc.ID.String() }>
                        @statusBadge(doc.Status)
                    </div>
                }
                @table.Cell() { @docActions(doc) }
            }
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Alpine.js for interactivity | Per-component vanilla JS | templUI v1.4.0 | Smaller bundle, no Alpine dependency |
| templui.min.js global bundle | Individual component scripts | templUI v1.4.0 | Better tree-shaking, load only what's needed |
| Empty props `Component()` | Optional variadic props | templUI pattern | Cleaner API for default usage |
| Manual dark mode classes | CSS variable-based theming | templUI design system | Automatic dark mode support |

**Current templUI patterns:**
- Per-component JavaScript loaded via `@component.Script()`
- CSS variables for theming (already configured in project)
- ARIA attributes for accessibility built into components
- Tailwind v4 with oklch colors

## Open Questions

Things that couldn't be fully resolved:

1. **PDF Viewer Modal Refactor**
   - What we know: Current implementation uses custom modal with PDF.js
   - What's unclear: Whether templUI dialog/sheet is suitable for full-screen PDF viewing
   - Recommendation: Evaluate during implementation - dialog for simple modal, sheet for slide-out panel, or keep custom if neither fits

2. **SelectBox HTMX Integration**
   - What we know: SelectBox uses custom JavaScript events
   - What's unclear: Exact event names for HTMX trigger integration
   - Recommendation: Test during implementation, may need `hx-trigger="change"` or custom event listener

3. **Sidebar Mobile Behavior**
   - What we know: templUI sidebar uses sheet component for mobile
   - What's unclear: How it integrates with existing mobile menu button
   - Recommendation: Follow templUI sidebar examples, may need to adjust header component

4. **Loading States Approach**
   - What we know: templUI has skeleton component
   - What's unclear: Best pattern for HTMX loading indicators vs skeleton placeholders
   - Recommendation: Use skeleton for initial page loads, keep `htmx-indicator` pattern for in-place updates

## Sources

### Primary (HIGH confidence)
- templUI GitHub repository - component source code, v1.4.0
- templUI CLI `templui list` - verified component availability
- Existing project code - verified patterns in login.templ, document_detail.templ

### Secondary (MEDIUM confidence)
- templUI documentation at templui.io/docs/components
- Raw GitHub component files (table.templ, sidebar.templ, badge.templ, etc.)

### Tertiary (LOW confidence)
- WebSearch results for templUI HTMX integration patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Verified via templUI CLI and existing project usage
- Architecture: HIGH - Based on existing project patterns and templUI source
- Pitfalls: MEDIUM - Based on common templ/HTMX issues and templUI docs

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable library)
