# Phase 14: Production Readiness - Research

**Researched:** 2026-02-04
**Domain:** Production deployment, security auditing, documentation
**Confidence:** HIGH

## Summary

This phase prepares Docko for production deployment through four distinct deliverables: comprehensive README documentation, production Docker Compose configuration, git history security audit, and complete .gitignore coverage. Research covers Docker Compose production patterns (health checks, resource limits, logging), secret scanning tools (gitleaks), history rewriting (git-filter-repo), and documentation best practices.

The project already has a `/health` endpoint implemented as a simple HTTP 200 response. The existing Docker Compose file provides a good foundation with the postgres healthcheck pattern already in place. The current .gitignore covers most essentials but needs expansion for comprehensive coverage.

**Primary recommendation:** Create standalone `docker-compose.prod.yml` with explicit health checks, resource limits, and JSON logging; run gitleaks to scan git history (IMPORTANT: real API key found in .envrc); expand .gitignore and create comprehensive README with deployment, backup, and troubleshooting guidance.

## Standard Stack

The established tools for this domain:

### Core Tools
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| [gitleaks](https://github.com/gitleaks/gitleaks) | v8.19+ | Git history secret scanning | Industry standard SAST tool, uses regex + entropy analysis |
| [git-filter-repo](https://github.com/newren/git-filter-repo) | v2.47+ | History rewriting if secrets found | Recommended by git core developers, 10x faster than filter-branch |
| Docker Compose | v2.x | Production orchestration | Native support for health checks, resource limits, logging |

### Supporting Tools
| Tool | Purpose | When to Use |
|------|---------|-------------|
| pg_dump | PostgreSQL backup | Database backup/restore procedures |
| docker cp | Volume backup | Document storage backup |
| openssl | Secret generation | Generate SESSION_SECRET, CREDENTIAL_ENCRYPTION_KEY |

### Installation Commands
```bash
# gitleaks
brew install gitleaks          # macOS
# or
docker pull ghcr.io/gitleaks/gitleaks:latest

# git-filter-repo (only if secrets found)
brew install git-filter-repo   # macOS
pip install git-filter-repo    # cross-platform
```

## Architecture Patterns

### Production Docker Compose Structure
```yaml
# docker-compose.prod.yml - STANDALONE, no extends
services:
  app:
    image: docko:latest
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      - DATABASE_URL=postgres://...
      # All config via environment
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "5"
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - docko-network
    volumes:
      - docko-storage:/app/storage

  postgres:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U docko -d docko"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
    networks:
      - docko-network
    volumes:
      - docko-postgres-data:/var/lib/postgresql/data

  ocrmypdf:
    # Similar pattern with health check, limits, logging
    ...

networks:
  docko-network:
    name: docko-network

volumes:
  docko-postgres-data:
  docko-storage:
```

### Health Check Implementation Options

The existing health endpoint is minimal:
```go
func (h *Handler) Health(c echo.Context) error {
    return c.String(http.StatusOK, "OK")
}
```

**Option A: Keep simple (recommended for this phase)**
- Current implementation sufficient for Docker health checks
- Returns 200 OK when app is running
- Fast, low overhead

**Option B: Enhanced with dependency checks (future consideration)**
```go
func (h *Handler) Health(c echo.Context) error {
    // Check database connectivity
    if err := h.db.Ping(c.Request().Context()); err != nil {
        return c.JSON(http.StatusServiceUnavailable, map[string]string{
            "status": "unhealthy",
            "error":  "database unreachable",
        })
    }
    return c.JSON(http.StatusOK, map[string]string{
        "status": "healthy",
    })
}
```

**Recommendation:** Keep simple for this phase. Health checks run every 30s; database ping adds latency and potential connection pool pressure.

### README Structure Pattern
```markdown
# Docko

[Brief description]

## Quick Start (Development)
[Fastest path to running locally]

## Production Deployment

### Prerequisites
[Required software, versions]

### Configuration
[Complete environment variable reference]

### Docker Compose Deployment
[Step-by-step production setup]

## Backup & Restore

### Database Backup
[pg_dump commands]

### Database Restore
[pg_restore commands]

### Document Storage Backup
[Volume backup procedure]

## Upgrade Procedures
[How to update, migration handling]

## Troubleshooting
[Common issues and solutions]
```

### Anti-Patterns to Avoid
- **Using `latest` tag in production:** Always pin specific versions for reproducibility
- **Sharing dev and prod compose:** Production should be completely standalone
- **Missing restart policies:** Containers should restart on failure
- **Unbounded logs:** Will fill disk without max-size limits
- **Hardcoded secrets:** All secrets via environment variables

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Secret scanning | grep for "password" | gitleaks | Handles 100+ secret patterns, entropy analysis, git history traversal |
| History rewriting | git filter-branch | git-filter-repo | 10x faster, maintained, recommended by git |
| Log rotation | Custom logrotate | Docker json-file driver | Built-in, per-container, compressed |
| Health checks | Custom monitoring | Docker HEALTHCHECK | Native container orchestration support |

**Key insight:** Production deployment patterns are well-established. Docker Compose v2 has all needed features built-in. Security tools like gitleaks cover far more patterns than manual regex searches.

## Common Pitfalls

### Pitfall 1: Secrets in Git History
**What goes wrong:** Even after removing secrets from current files, they remain in git history forever.
**Why it happens:** Git stores complete history; `rm` doesn't remove past commits.
**How to avoid:**
1. Run gitleaks before every release
2. If found: revoke secret immediately, then optionally rewrite history
3. Add proper .gitignore BEFORE creating secrets
**Warning signs:** gitleaks output shows matches in git log

### Pitfall 2: Health Check Tool Availability
**What goes wrong:** Health check fails because curl/wget not in container image.
**Why it happens:** Alpine-based images are minimal, don't include network tools.
**How to avoid:**
- Option A: Install wget in Dockerfile (`apk add --no-cache wget`)
- Option B: Use native health check binary if available
- Option C: Use Go binary that responds to health endpoint
**Warning signs:** Health check shows "exec: curl: not found" in docker logs

### Pitfall 3: Resource Limits Breaking App
**What goes wrong:** Container killed by OOM or throttled to unusable state.
**Why it happens:** Limits set too low for actual workload.
**How to avoid:**
- Start with generous limits (512M RAM, 1 CPU)
- Monitor actual usage with `docker stats`
- Adjust based on production data
**Warning signs:** Container exits with code 137 (OOM killed)

### Pitfall 4: Database Connection String Exposure
**What goes wrong:** DATABASE_URL with password visible in docker-compose.yml committed to repo.
**Why it happens:** Convenience of having all config in one place.
**How to avoid:**
- Use environment variable substitution: `${DATABASE_URL}`
- Or Docker secrets for swarm deployments
- Document that users must set env vars, don't provide defaults with real passwords

### Pitfall 5: Volume Permissions
**What goes wrong:** App can't write to mounted volumes.
**Why it happens:** Container user doesn't match host user permissions.
**How to avoid:**
- Use named volumes (Docker manages permissions)
- Or specify user in compose: `user: "1000:1000"`
**Warning signs:** "permission denied" errors in app logs

### Pitfall 6: Log Rotation Not Working
**What goes wrong:** Disk fills up despite configuring max-size.
**Why it happens:** Existing containers don't pick up new logging config.
**How to avoid:** After changing logging config, recreate containers: `docker compose down && docker compose up -d`
**Warning signs:** Log files larger than max-size setting

## Code Examples

### gitleaks Commands

**Scan entire git history:**
```bash
# Full scan with verbose output
gitleaks git -v

# Generate JSON report
gitleaks git --report-format json --report-path gitleaks-report.json

# Scan specific branch
gitleaks git --log-opts="main"
```

**Scan current directory (no git history):**
```bash
gitleaks dir -v .
```

**Using Docker:**
```bash
docker run --rm -v $(pwd):/repo ghcr.io/gitleaks/gitleaks:latest git -v --source=/repo
```

### git-filter-repo Commands (if secrets found)

**Remove file containing secret from all history:**
```bash
# BACKUP FIRST
git clone --bare . ../docko-backup.git

# Remove file from history
git filter-repo --invert-paths --path .envrc

# Or replace specific text within files
echo 'sk-proj-YOUR_ACTUAL_KEY==>REDACTED' > expressions.txt
git filter-repo --replace-text expressions.txt
```

**Force push after rewriting:**
```bash
git remote add origin <url>
git push --force --all
git push --force --tags
```

**Team must re-clone after history rewrite:**
```bash
# Everyone else runs:
git fetch origin
git reset --hard origin/main
# Do NOT merge - will reintroduce old history
```

### pg_dump Backup/Restore

**Backup:**
```bash
# From host (if postgres exposed)
pg_dump -h localhost -U docko -d docko > backup.sql

# From Docker container
docker exec docko-postgres pg_dump -U docko docko > backup.sql

# Compressed
docker exec docko-postgres pg_dump -U docko docko | gzip > backup.sql.gz
```

**Restore:**
```bash
# Plain SQL
docker exec -i docko-postgres psql -U docko docko < backup.sql

# Compressed
gunzip -c backup.sql.gz | docker exec -i docko-postgres psql -U docko docko
```

### Volume Backup/Restore

**Backup document storage:**
```bash
# Create tarball of named volume
docker run --rm \
  -v docko-storage:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/storage-backup.tar.gz -C /data .
```

**Restore:**
```bash
docker run --rm \
  -v docko-storage:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/storage-backup.tar.gz"
```

### Docker Compose Health Check Patterns

**HTTP endpoint check:**
```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

**PostgreSQL ready check:**
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U docko -d docko"]
  interval: 10s
  timeout: 5s
  retries: 5
```

### Comprehensive .gitignore Additions

```gitignore
# === Existing patterns (keep) ===
# Binaries, Build artifacts, Environment, Generated files, etc.

# === Additional patterns needed ===

# Secrets and credentials
*.pem
*.key
*.crt
*.p12
*.pfx
credentials.json
secrets.json
.secrets

# Backup files
*.bak
*.backup
*.sql
*.dump
backup/
backups/

# Test artifacts
coverage/
*.coverprofile
profile.cov
coverage.*

# Log files (beyond tmp/)
*.log
logs/

# Editor/IDE (expand existing)
*.swp
*.swo
*~
.idea/
.vscode/
*.sublime-*

# OS files (expand existing)
.DS_Store
Thumbs.db
Desktop.ini
.Spotlight-V100
.Trashes

# Go specific
*.test
*.out
go.work
go.work.sum

# Docker
.docker/

# Planning docs (if not committed)
# .planning/

# Local development overrides
docker-compose.override.yml
*.local.yml
*.local.yaml
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| git filter-branch | git-filter-repo | ~2019 | 10x faster, maintained, recommended |
| Manual secret search | gitleaks/trufflehog | ~2020 | Automated, 100+ patterns, CI integration |
| External log rotation | Docker json-file driver | Docker 1.8+ | Built-in, per-container |
| docker-compose v1 | docker compose v2 | 2021 | Integrated into Docker CLI, better health checks |

**Deprecated/outdated:**
- `git filter-branch`: Slow, error-prone, officially deprecated
- `docker-compose` (standalone): Replaced by `docker compose` plugin
- BFG Repo-Cleaner: Still works but git-filter-repo preferred

## Open Questions

### 1. Health Check Complexity
**What we know:** Simple 200 OK response is sufficient for Docker health checks.
**What's unclear:** Whether to add database ping for "true" health status.
**Recommendation:** Keep simple for this phase. Document enhanced option for future.

### 2. OCRmyPDF Health Check
**What we know:** OCRmyPDF runs as a file watcher, not HTTP server.
**What's unclear:** How to verify it's healthy without an HTTP endpoint.
**Recommendation:** Use process-based check or skip health check (it has restart: unless-stopped).

### 3. Secrets Already in History
**What we know:** Real OpenAI API key exists in current .envrc (not committed, in .gitignore).
**What's unclear:** Whether any secrets were ever committed to history.
**Recommendation:** Run gitleaks scan as first action; if found, revoke and optionally rewrite.

## Sources

### Primary (HIGH confidence)
- [Docker JSON File Logging Driver Docs](https://docs.docker.com/engine/logging/drivers/json-file/) - Logging configuration
- [Docker Health Check Docs](https://docs.docker.com/reference/dockerfile/#healthcheck) - Health check syntax
- [gitleaks GitHub Repository](https://github.com/gitleaks/gitleaks) - Installation, commands, configuration
- [git-filter-repo GitHub Repository](https://github.com/newren/git-filter-repo) - Installation, usage
- [GitHub gitignore Templates - Go](https://github.com/github/gitignore/blob/main/Go.gitignore) - Go patterns
- [GitHub gitignore Templates - Node](https://github.com/github/gitignore/blob/main/Node.gitignore) - Node patterns

### Secondary (MEDIUM confidence)
- [Docker Compose Health Checks Guide](https://last9.io/blog/docker-compose-health-checks/) - Best practices
- [Docker Best Practices 2026](https://thinksys.com/devops/docker-best-practices/) - Resource limits patterns
- [GitHub Docs - Removing Sensitive Data](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository) - History rewriting workflow
- [SimpleBackups Docker Postgres Guide](https://simplebackups.com/blog/docker-postgres-backup-restore-guide-with-examples) - Backup/restore procedures

### Tertiary (LOW confidence)
- WebSearch results for README best practices - General guidance

## Metadata

**Confidence breakdown:**
- Docker Compose patterns: HIGH - Official Docker documentation
- gitleaks usage: HIGH - Official repository documentation
- git-filter-repo usage: HIGH - Official repository and GitHub docs
- .gitignore patterns: HIGH - Official GitHub templates
- README structure: MEDIUM - Multiple community sources agree
- Resource limit values: MEDIUM - General guidance, needs tuning per workload

**Research date:** 2026-02-04
**Valid until:** 2026-03-04 (30 days - stable domain)

## Critical Finding

**IMPORTANT:** During research, a real OpenAI API key was observed in `/home/bjk/projects/corpus/docko/.envrc`:
```
export OPENAI_API_KEY=sk-proj-WQxf...
```

This file IS in .gitignore, so it should not be committed. However:
1. Run `gitleaks git -v` to verify no secrets in history
2. If this key was ever committed, it MUST be revoked immediately
3. The .envrc.example correctly shows placeholder values

This validates the importance of the security audit in this phase.
