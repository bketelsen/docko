# docko

Go web application with Echo, Templ, HTMX, and Tailwind CSS.

## Critical: Check Build Logs

**ALWAYS check `./tmp/air-combined.log` after making code changes.**

This log contains:
- Compilation errors
- Template generation errors
- SQL generation errors

Never assume code changes succeeded without checking this log.

## Development Workflow

`make dev` is always running during development. It automatically:
1. Kills existing process on PORT
2. Regenerates Templ templates
3. Regenerates sqlc queries
4. Runs `go mod tidy`
5. Rebuilds and restarts the server

**You do NOT need to manually run:** `templ generate`, `sqlc generate`, `go build`, or `air`

## Quick Start (Development)

```bash
# Start PostgreSQL
docker compose up -d

# Load environment (already configured for docker-compose defaults)
direnv allow

# Start the server with hot reload
make dev
```

## Environment

All config via `.envrc` with direnv:

```bash
DATABASE_URL    # PostgreSQL connection string (required)
PORT            # Server port (default: 3000)
ENV             # development | production
LOG_LEVEL       # DEBUG | INFO | WARN | ERROR
SITE_NAME       # Used in templates and meta tags
SITE_URL        # Base URL for canonical links
```

Default `.envrc` is pre-configured for `docker compose up -d`.

## Key Commands

| Command | What it does |
|---------|--------------|
| `docker compose up -d` | Start PostgreSQL for development |
| `docker compose down` | Stop PostgreSQL |
| `make dev` | Start with hot reload (main workflow) |
| `make build` | Build production binary |
| `make test` | Run tests with race detection |
| `make lint` | Run golangci-lint and templ fmt |
| `make generate` | Regenerate templ + sqlc code |
| `make migrate` | Run database migrations |
| `make migrate-create NAME=xxx` | Create new migration |
| `make css-watch` | Watch Tailwind (separate terminal) |
| `make setup` | Install dev tools |

## Project Structure

```
cmd/server/          Entry point, slog config, generate directives
internal/
  config/            Environment configuration
  ctxkeys/           Typed context keys
  database/          Database connection, migrations, sqlc
  handler/           HTTP handlers
  meta/              SEO/OG metadata helpers
  middleware/        Echo middleware
  testutil/          Test helpers
templates/
  layouts/           Base layouts (base.templ, admin.templ)
  pages/             Page templates
  components/        Reusable components
static/
  css/               Tailwind input/output
  js/                JavaScript files
  images/            Static images
sqlc/
  sqlc.yaml          SQLC configuration
  queries/           SQL query files
```

## Code Patterns

### Logging
Use `slog` (never `fmt.Printf` or `log.Printf`):
```go
slog.Info("message", "key", value)
slog.Error("failed to X", "error", err)
```

### Errors
Wrap with context:
```go
return fmt.Errorf("failed to create user: %w", err)
```

### Database
Use sqlc-generated queries in `internal/database/sqlc/`:
```go
user, err := h.db.Queries.GetUser(ctx, userID)
```

### Templates
Templates construct their own meta - handlers don't pass it:
```go
// Handler - just render the template
func (h *Handler) Home(c echo.Context) error {
    return pages.Home().Render(c.Request().Context(), c.Response().Writer)
}
```

```templ
// Template - constructs its own meta
templ Home() {
    @layouts.Base(meta.New("Home", "Description")) {
        // content
    }
}
```

### Admin Dashboard
Uses custom layouts with dark mode support. Theme toggle persists to localStorage.

## Database

PostgreSQL with:
- `pgx/v5` driver
- `goose` migrations (embedded in binary)
- `sqlc` for type-safe queries

Migrations run automatically on startup.

## Testing

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...
```

Tests require `TEST_DATABASE_URL` or `DATABASE_URL` environment variable.
