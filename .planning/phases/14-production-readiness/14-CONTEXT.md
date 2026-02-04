# Phase 14: Production Readiness - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Prepare Docko for production deployment. Deliverables: comprehensive README with setup/config/deployment instructions, production Docker Compose file, git history audit for secrets, and complete .gitignore coverage. No new features — documentation and security hardening only.

</domain>

<decisions>
## Implementation Decisions

### Docker Compose Design
- Separate production file: `docker-compose.prod.yml` (independent from dev compose)
- No reverse proxy included — assume external proxy (nginx, load balancer) handles SSL
- HTTP endpoint health checks — add `/health` route to app, compose checks it
- Named volumes for persistent data (database, document storage)
- Restart policy: `unless-stopped`
- Include resource limits with reasonable defaults (mem_limit, cpu examples)
- JSON file logging with max-size and max-file to prevent disk fill
- Custom named network for explicit service connections

### Security Audit Approach
- Use **gitleaks** for git history secret scanning
- If secrets found: rewrite history using git-filter-repo, force push
- Comprehensive manual code review for hardcoded secrets (password, secret, key, token patterns)
- Verify no test credentials, example passwords, or API keys in source

### Deployment Guidance (README)
- Primary target: Self-hosted Docker on VPS/bare metal
- Detailed backup/restore procedures for database (pg_dump) and document storage volumes
- Upgrade procedures with migration notes — how to pull updates, run migrations, handle breaking changes
- Troubleshooting section covering common issues: connection problems, permission errors, typical gotchas

### Claude's Discretion
- .gitignore completeness categories — determine appropriate coverage based on project structure
- README section ordering and depth
- Specific resource limit values
- Health check endpoint implementation details

</decisions>

<specifics>
## Specific Ideas

- Production compose should be completely standalone — no extends, no dev dependencies
- Health endpoint at `/health` (common convention)
- Document storage uses same volume approach as OCR integration already established

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 14-production-readiness*
*Context gathered: 2026-02-04*
