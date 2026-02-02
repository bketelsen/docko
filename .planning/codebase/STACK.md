# Technology Stack

**Analysis Date:** 2026-02-02

## Languages

**Primary:**
- Go 1.23.0 - Server-side application, all backend logic

**Secondary:**
- HTML/CSS - Template rendering via Templ
- TypeScript/JavaScript - Frontend components and Tailwind CLI
- SQL - Database queries (PostgreSQL)

## Runtime

**Environment:**
- Go 1.23.0 (Linux/Unix)

**Package Manager:**
- `go mod` for Go dependencies
- `npm` for Node.js/Tailwind tooling
- Lockfiles: `go.sum` (present), `package-lock.json` (present)

## Frameworks

**Core:**
- Echo 4.13.3 - HTTP web framework (`github.com/labstack/echo/v4`)
- Templ 0.3.977 - Type-safe HTML templating (`github.com/a-h/templ`)

**Database:**
- pgx/v5 5.7.2 - PostgreSQL driver (`github.com/jackc/pgx/v5`)
- sqlc - Type-safe SQL code generation (via `sqlc generate`, queries in `sqlc/queries/`)
- Goose 3.24.1 - Database migration tool (`github.com/pressly/goose/v3`)

**Frontend UI:**
- Tailwind CSS 4.0.0 - Utility-first CSS framework
- HTMX - JavaScript interactions (via unpkg CDN, loaded in middleware CSP)
- templUI - Pre-built component library (`components/` directory)

**Styling:**
- Tailwind CSS CLI 4.0.0 - CSS compilation (`@tailwindcss/cli`)
- tailwind-merge-go 0.2.1 - Merge Tailwind class names (`github.com/Oudwins/tailwind-merge-go`)

**Development:**
- Air - Hot reload for development (`go install github.com/air-verse/air@latest`)

## Key Dependencies

**Critical:**
- `github.com/jackc/pgx/v5` - PostgreSQL connection pooling and type-safe queries
- `github.com/labstack/echo/v4` - HTTP routing and middleware
- `github.com/a-h/templ` - Template generation and rendering
- `github.com/pressly/goose/v3` - Embedded database migrations

**Infrastructure:**
- `golang.org/x/crypto` 0.40.0 - Cryptographic functions (bcrypt for password hashing)
- `github.com/google/uuid` 1.6.0 - UUID generation for database IDs
- `github.com/lmittmann/tint` 1.0.6 - Colored logging output for development

**Logging:**
- `log/slog` (Go stdlib) - Structured logging with custom handler configuration

## Configuration

**Environment:**
- Configuration via `.envrc` (direnv) with sensible defaults
- Environment variables:
  - `DATABASE_URL` (required) - PostgreSQL connection string
  - `PORT` (default: 3000) - Server port
  - `ENV` (default: development) - Environment mode
  - `LOG_LEVEL` (default: INFO) - Logging verbosity
  - `SITE_NAME` (default: docko) - Site title
  - `SITE_URL` (default: http://localhost:3000) - Base URL for canonical links
  - `ADMIN_PASSWORD` (required) - Admin user password
  - `SESSION_SECRET` (auto-generated in dev) - HMAC secret for sessions
  - `SESSION_MAX_AGE` (default: 24) - Session expiration in hours

**Build:**
- `.air.toml` - Air hot reload configuration
- `Makefile` - Build targets and development commands
- `cmd/server/generate.go` - Go generate directives for sqlc, templ, Tailwind
- `cmd/server/slog.go` - Structured logging setup (tint for dev, JSON for prod)

**Database:**
- `sqlc/sqlc.yaml` - SQLC code generation config
- `internal/database/migrations/` - Embedded SQL migrations (Goose format)

## Platform Requirements

**Development:**
- Go 1.23.0+
- PostgreSQL 16+ (via Docker Compose)
- Node.js 18+ (for Tailwind CLI)
- Linux/macOS/Windows with bash

**Production:**
- Linux binary (compiled with `go build`)
- PostgreSQL database (connection string via `DATABASE_URL`)
- Deployment target: Any platform supporting Linux executables (Docker, systemd, etc.)

## Tools

**CLI/Build Tools:**
- `templ` - Template generation from `.templ` files
- `sqlc` - SQL to Go code generation from SQL queries
- `goose` - Database migration runner
- `air` - Live reload development server
- `golangci-lint` - Go linting (via `make lint`)
- `@tailwindcss/cli` - Tailwind CSS compilation

**Development Workflow:**
- `make dev` - Runs air with automatic templ/sqlc/CSS regeneration
- `make generate` - Runs `go generate` (templ, sqlc, Tailwind)
- `docker compose up -d` - Starts PostgreSQL development database

---

*Stack analysis: 2026-02-02*
