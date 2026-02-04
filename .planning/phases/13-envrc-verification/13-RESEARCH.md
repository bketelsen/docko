# Phase 13: Environment Configuration Verification - Research

**Researched:** 2026-02-03
**Domain:** Environment configuration, direnv, documentation
**Confidence:** HIGH

## Summary

This phase is a documentation/maintenance task focused on ensuring all environment variables used by the docko application are properly documented in `.envrc.example`. The research involved auditing the codebase to identify all `os.Getenv()` calls and comparing them against the existing documentation.

The codebase uses two environment configuration files: `.envrc` (actual values, gitignored) and `.envrc.example` (template with documentation). The current `.envrc.example` is significantly incomplete - it documents only 8 variables while the codebase uses 23+ distinct environment variables.

**Primary recommendation:** Update `.envrc.example` to document all environment variables with categorized sections, descriptions, and example values.

## Standard Stack

This phase doesn't require new libraries - it's purely documentation work.

### Core Tools Used
| Tool | Purpose | Why Standard |
|------|---------|--------------|
| direnv | Automatic environment variable loading | Standard for per-project env management |
| grep/search | Audit codebase for env vars | Standard development tools |

### File Conventions
| File | Purpose | Git Status |
|------|---------|------------|
| `.envrc` | Actual development values | SHOULD be gitignored (contains secrets) |
| `.envrc.example` | Template with documentation | SHOULD be committed |

## Architecture Patterns

### Environment Variable Organization

**Pattern: Categorized Sections**
Group related environment variables together with clear section headers:

```bash
# =============================================================================
# Database
# =============================================================================
export DATABASE_URL="..."

# =============================================================================
# Server
# =============================================================================
export PORT="..."
```

**Pattern: Description Comments**
Each variable should have a comment explaining:
1. What it does
2. Whether it's required or optional
3. Default value if optional
4. Valid values/format

```bash
# Admin login password (REQUIRED for auth, CHANGE IN PRODUCTION)
export ADMIN_PASSWORD="admin123"

# HMAC secret for sessions (optional - auto-generated in development)
# Generate with: openssl rand -base64 32
# export SESSION_SECRET="your-32-char-secret"
```

### Project Structure

The environment documentation should mirror the code organization:
```
.envrc.example
├── Database section       -> internal/config/config.go
├── Server section         -> internal/config/config.go
├── Site/SEO section       -> internal/config/config.go
├── Authentication section -> internal/config/config.go
├── Storage section        -> internal/config/config.go
├── Inbox section          -> internal/config/config.go
├── Network section        -> internal/config/config.go
├── AI Providers section   -> internal/ai/*.go
└── Testing section        -> internal/testutil/, tests
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Environment loading | Custom loader | direnv | Already in use, shell-native |
| Secret generation | Manual strings | `openssl rand -base64 32` | Cryptographically secure |

## Common Pitfalls

### Pitfall 1: Committing Secrets
**What goes wrong:** `.envrc` with real API keys gets committed to git
**Why it happens:** Developer forgets to gitignore or uses wrong file
**How to avoid:**
- Ensure `.envrc` is in `.gitignore`
- Only commit `.envrc.example` with placeholder values
- Never put real API keys in example files
**Warning signs:** Git diff shows API keys, `git status` shows `.envrc` as tracked

### Pitfall 2: Outdated Documentation
**What goes wrong:** New env vars added to code but not to `.envrc.example`
**Why it happens:** Documentation update forgotten during feature development
**How to avoid:**
- Include `.envrc.example` updates in PR checklist
- Periodic audit like this phase
**Warning signs:** Features fail silently due to missing env vars

### Pitfall 3: Missing Required vs Optional Distinction
**What goes wrong:** Users don't know which vars are required
**Why it happens:** Comments don't clearly indicate requirement level
**How to avoid:**
- Use consistent marking: (REQUIRED), (optional), (auto-generated)
- Document defaults explicitly
**Warning signs:** App crashes on startup with unclear env var errors

### Pitfall 4: Inconsistent Placeholder Values
**What goes wrong:** Example values cause confusion or security issues
**Why it happens:** Using realistic-looking fake values that might be mistaken for real
**How to avoid:**
- Use obviously placeholder values: `your-api-key-here`, `changeme123`
- Avoid real-looking but invalid API keys
**Warning signs:** Test failures from accidentally using placeholder values

## Code Examples

### Current vs Required Documentation

**Current `.envrc.example` (incomplete):**
```bash
# Copy this file to .envrc and edit with your values
# Then run: direnv allow

# Database (PostgreSQL)
export DATABASE_URL="postgres://docko:docko@localhost:5432/docko?sslmode=disable"

# Server
export PORT="3000"
export ENV="development"
export LOG_LEVEL="DEBUG"

# Site / SEO
export SITE_NAME="docko"
export SITE_URL="http://localhost:3000"
export DEFAULT_OG_IMAGE="/static/images/og-default.png"
```

**Required additions (identified from codebase audit):**

```bash
# =============================================================================
# Authentication
# =============================================================================
# Admin login password (REQUIRED for auth - CHANGE IN PRODUCTION)
export ADMIN_PASSWORD="changeme123"

# Session HMAC secret (optional - auto-generated in development)
# Generate with: openssl rand -base64 32
# export SESSION_SECRET="your-32-char-secret-here"

# Session max age in hours (optional, default: 24)
# export SESSION_MAX_AGE="24"

# =============================================================================
# Storage
# =============================================================================
# Root path for document storage (optional, default: ./storage)
# export STORAGE_PATH="./storage"

# =============================================================================
# Inbox (Document Ingestion)
# =============================================================================
# Default inbox path for auto-import (optional - disabled if not set)
# export INBOX_PATH="/path/to/inbox"

# Subdirectory for files that fail processing (optional, default: errors)
# export INBOX_ERROR_SUBDIR="errors"

# Maximum file size in MB (optional, default: 100)
# export INBOX_MAX_FILE_SIZE_MB="100"

# Directory scan interval in milliseconds (optional, default: 1000)
# export INBOX_SCAN_INTERVAL_MS="1000"

# =============================================================================
# Network Sources (SMB/NFS)
# =============================================================================
# Encryption key for storing network source credentials (REQUIRED for network sources)
# Generate with: openssl rand -base64 32
# export CREDENTIAL_ENCRYPTION_KEY="your-32-char-key-here"

# =============================================================================
# AI Providers (all optional - enable one or more)
# =============================================================================
# OpenAI API key for document analysis
# export OPENAI_API_KEY="sk-your-openai-key"

# Anthropic API key for document analysis
# export ANTHROPIC_API_KEY="sk-ant-your-anthropic-key"

# Ollama host URL (optional, defaults to localhost:11434)
# export OLLAMA_HOST="http://localhost:11434"

# Ollama model name (optional, default: llama3.2)
# export OLLAMA_MODEL="llama3.2"

# =============================================================================
# Testing (not needed for normal development)
# =============================================================================
# Separate database URL for tests (uses DATABASE_URL if not set)
# export TEST_DATABASE_URL="postgres://docko:docko@localhost:5432/docko_test?sslmode=disable"

# Path to PDF file for PDF extraction tests
# export TEST_PDF_WITH_TEXT="/path/to/test.pdf"
```

## Environment Variable Inventory

### Complete list from codebase audit:

| Variable | Source File | Required | Default | Purpose |
|----------|-------------|----------|---------|---------|
| DATABASE_URL | config.go:51 | YES | - | PostgreSQL connection string |
| PORT | config.go:52 | no | "3000" | Server port |
| ENV | config.go:53, slog.go:14 | no | "development" | Environment mode |
| LOG_LEVEL | slog.go:34 | no | INFO | Logging level |
| SITE_NAME | config.go:55 | no | "docko" | Site name for meta tags |
| SITE_URL | config.go:56 | no | "http://localhost:3000" | Base URL for canonical links |
| DEFAULT_OG_IMAGE | config.go:57 | no | "/static/images/og-default.png" | Default OG image |
| ADMIN_PASSWORD | config.go:60 | YES* | - | Admin login password |
| SESSION_SECRET | config.go:61 | no | auto-generated | Session HMAC secret |
| SESSION_MAX_AGE | config.go:62 | no | 24 | Session max age in hours |
| STORAGE_PATH | config.go:65 | no | "./storage" | Document storage root |
| INBOX_PATH | config.go:68 | no | - | Auto-import inbox path |
| INBOX_ERROR_SUBDIR | config.go:69 | no | "errors" | Error file subdirectory |
| INBOX_MAX_FILE_SIZE_MB | config.go:70 | no | 100 | Max file size for import |
| INBOX_SCAN_INTERVAL_MS | config.go:71 | no | 1000 | Directory scan interval |
| CREDENTIAL_ENCRYPTION_KEY | config.go:74 | YES** | - | Network credential encryption |
| OPENAI_API_KEY | openai.go:55 | no | - | OpenAI API access |
| ANTHROPIC_API_KEY | anthropic.go:20 | no | - | Anthropic API access |
| OLLAMA_HOST | ollama.go:34 | no | localhost:11434 | Ollama server URL |
| OLLAMA_MODEL | ollama.go:20 | no | "llama3.2" | Ollama model name |
| TEST_DATABASE_URL | testutil.go:18 | no | DATABASE_URL | Test database URL |
| TEST_PDF_WITH_TEXT | text_test.go:191 | no | - | PDF test file path |

*Required for admin authentication to work
**Required only if using network sources feature

### Variables in .envrc.example (current):
1. DATABASE_URL
2. PORT
3. ENV
4. LOG_LEVEL
5. SITE_NAME
6. SITE_URL
7. DEFAULT_OG_IMAGE

### Variables MISSING from .envrc.example:
1. ADMIN_PASSWORD
2. SESSION_SECRET
3. SESSION_MAX_AGE
4. STORAGE_PATH
5. INBOX_PATH
6. INBOX_ERROR_SUBDIR
7. INBOX_MAX_FILE_SIZE_MB
8. INBOX_SCAN_INTERVAL_MS
9. CREDENTIAL_ENCRYPTION_KEY
10. OPENAI_API_KEY
11. ANTHROPIC_API_KEY
12. OLLAMA_HOST
13. OLLAMA_MODEL
14. TEST_DATABASE_URL
15. TEST_PDF_WITH_TEXT

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Flat .env files | Structured .envrc with direnv | ongoing | Better shell integration, auto-load |
| Required vars only | All vars documented | best practice | Self-documenting, easier onboarding |

**Best practices from [direnv documentation](https://direnv.net/):**
- Use `.envrc.example` committed to git as template
- Keep `.envrc` gitignored with actual values
- Group related variables with section headers
- Include comments for each variable

## Open Questions

None - this is a straightforward documentation task with no unresolved technical questions.

## Sources

### Primary (HIGH confidence)
- Codebase audit via grep for `os.Getenv`, `getEnvOrDefault`, `getEnvIntOrDefault`
- `/home/bjk/projects/corpus/docko/internal/config/config.go` - Central configuration loading
- `/home/bjk/projects/corpus/docko/internal/ai/*.go` - AI provider configuration
- `/home/bjk/projects/corpus/docko/.envrc` - Current development values
- `/home/bjk/projects/corpus/docko/.envrc.example` - Current template

### Secondary (MEDIUM confidence)
- [direnv official documentation](https://direnv.net/) - Best practices for .envrc files
- [direnv best practices blog](https://dev.to/allenap/some-direnv-best-practices-actually-just-one-4864)

## Metadata

**Confidence breakdown:**
- Environment variable inventory: HIGH - Direct codebase audit
- Documentation patterns: HIGH - Industry standard practices
- Best practices: MEDIUM - Community sources verified against official docs

**Research date:** 2026-02-03
**Valid until:** 90 days (documentation patterns are stable)

## Implementation Notes

This phase is straightforward:
1. Task 1: Audit and verify all env vars (this research already done)
2. Task 2: Update `.envrc.example` with all missing variables
3. Task 3: Verify `.envrc` is properly gitignored
4. Task 4: Verify current `.envrc` has all documented variables

The research has already identified all 15 missing variables. The planner can create tasks to systematically add them to `.envrc.example` in categorized sections.
