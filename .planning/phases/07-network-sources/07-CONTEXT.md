# Phase 7: Network Sources - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Auto-import documents from SMB and NFS network shares. Users configure sources with credentials, system monitors for new files and imports them. Management happens through the existing Inbox page. AI-powered tagging is a separate phase.

</domain>

<decisions>
## Implementation Decisions

### Connection setup
- Manual path entry only (no network browsing)
- Per-source credentials (username/password stored per source)
- Explicit protocol dropdown (user selects SMB or NFS)
- "Test Connection" button validates path and credentials before saving

### Import behavior
- User chooses per-source: continuous watch or manual-only sync
- Post-import action configurable per-source: leave in place, delete, or move to subfolder
- Import once only — modifications on source are ignored after initial import
- Fixed polling interval for continuous watch (e.g., 5 min, not user-configurable)
- Same duplicate_action options as local inboxes (delete/rename/skip)
- Configurable batch size limit per source
- Always recursive scanning of subfolders
- PDF files only (hardcoded, no pattern configuration)

### Source management UI
- Extend existing Inbox page (new tab or section for network sources)
- Detailed status display: files imported count, errors, connection state, last sync time
- Per-source "Sync Now" button plus global "Sync All" button
- Modal dialog for add/edit (consistent with inbox management pattern)

### Error feedback
- Visual indicator (red badge/icon) on source in list, plus toast notifications for new failures
- Individual file import failures logged with viewable error log in UI
- Auto-disable sources after N consecutive connection failures, require manual re-enable
- Per-file retry option for failed imports

### Claude's Discretion
- Exact polling interval value
- Number of failures before auto-disable
- Error log retention policy
- Connection timeout values
- Batch size default

</decisions>

<specifics>
## Specific Ideas

- UI pattern should mirror existing Inbox management for consistency
- Same duplicate handling enum (delete/rename/skip) as inboxes

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-network-sources*
*Context gathered: 2026-02-03*
