# External Integrations

**Analysis Date:** 2026-02-02

## APIs & External Services

**None Detected**

No third-party API integrations (Stripe, SendGrid, etc.) are present in the codebase.

## Data Storage

**Databases:**
- PostgreSQL 16 (Alpine)
  - Connection: `DATABASE_URL` env var (format: `postgres://user:pass@host:5432/dbname?sslmode=disable`)
  - Client: `pgx/v5` (`github.com/jackc/pgx/v5`) with connection pooling
  - Query Generation: `sqlc` type-safe code generation
  - Schema Management: Goose migrations (embedded in binary at `internal/database/migrations/`)
  - Tables: `examples`, `admin_users`, `admin_sessions`

**File Storage:**
- Local filesystem only
  - Static assets: `static/` directory (CSS, JS, images)
  - Templates: `templates/` and `components/` directories
  - No cloud storage integration (S3, etc.)

**Caching:**
- None (in-memory session storage only, stored in database)

## Authentication & Identity

**Auth Provider:**
- Custom (in-house implementation)
  - Implementation: Session-based auth with bcrypt password hashing
  - Service: `internal/auth/auth.go`
  - Storage: `admin_users` and `admin_sessions` tables in PostgreSQL
  - Password: `ADMIN_PASSWORD` env var sets/updates admin password on startup
  - Session Tokens: HMAC-SHA256 hashed with `SESSION_SECRET` (auto-generated in dev)
  - Session Expiration: `SESSION_MAX_AGE` hours (default: 24)
  - Cookie: `admin_session` (HttpOnly, SameSite=Lax)

**Auth Middleware:**
- Custom middleware at `internal/middleware/auth.go`
  - Protected routes use `middleware.RequireAuth(h.auth)`
  - Validates session token against database
  - Redirects to `/login` on unauthorized access

## Monitoring & Observability

**Error Tracking:**
- None (no Sentry, Rollbar, etc.)

**Logs:**
- Structured logging via Go stdlib `log/slog`
- Handler: `tint` for development (colored, human-readable), JSON for production
- Output: stderr
- Levels: DEBUG, INFO, WARN, ERROR (configurable via `LOG_LEVEL` env var)
- Request logging: Middleware in `internal/middleware/middleware.go` logs method, URI, status, latency
- Implementation: `cmd/server/slog.go` sets up default logger on init

**Tracing:**
- None detected

**Metrics:**
- None detected

## CI/CD & Deployment

**Hosting:**
- Not pre-configured (application is deployable anywhere)
- Docker Compose provided for local development only (`docker-compose.yml`)

**CI Pipeline:**
- Not configured (no `.github/workflows/` present)

**Build Process:**
- `make build` - Compiles Go binary
- Automatic regeneration: templ, sqlc, Tailwind CSS (via `go generate`)

## Environment Configuration

**Required env vars (at startup):**
- `DATABASE_URL` - Must be set, application exits if missing

**Secrets location:**
- `.envrc` file (gitignored) for development
- Environment variables for production
- Critical secrets: `ADMIN_PASSWORD`, `SESSION_SECRET`

## Webhooks & Callbacks

**Incoming:**
- None detected

**Outgoing:**
- None detected

## Background Services

**Session Cleanup:**
- Goroutine in `cmd/server/main.go` runs hourly cleanup of expired sessions
- Calls `authService.CleanupExpiredSessions(ctx)` via ticker
- Logs warnings if cleanup fails but doesn't block server

## Security Headers

**Middleware:**
- CORS: Allows all origins (`AllowOrigins: []string{"*"}`)
- Gzip: Compression at level 5
- Secure Headers (via Echo middleware):
  - X-XSS-Protection: "1; mode=block"
  - X-Content-Type-Options: "nosniff"
  - X-Frame-Options: "SAMEORIGIN"
  - HSTS: Max age 31536000 seconds (1 year)
  - Content-Security-Policy: `default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline';`

---

*Integration audit: 2026-02-02*
