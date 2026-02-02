# Coding Conventions

**Analysis Date:** 2026-02-02

## Naming Patterns

**Files:**
- Package files: lowercase with underscores (e.g., `auth.go`, `config.go`)
- Handlers: `handler.go`, `auth.go`, `admin.go`
- Tests: `*_test.go` suffix
- Generated code: `*_templ.go` (Templ), `*.sql.go` (SQLC)

**Functions:**
- PascalCase for exported functions (e.g., `New()`, `RegisterRoutes()`, `LoginPage()`)
- camelCase for unexported functions (e.g., `clearSessionCookie()`, `getEnvOrDefault()`)
- Handler methods: use descriptive names matching their responsibility (e.g., `AdminDashboard()`, `ValidateSession()`)

**Variables:**
- camelCase for all variables and constants (e.g., `cfg`, `db`, `authService`)
- Constants: ALL_CAPS for package-level constants (e.g., `AdminUsername`, `TokenLength`, `SessionCookieName`)
- Short names for loop counters and temporary variables (e.g., `err`, `ctx`)

**Types:**
- PascalCase for struct names (e.g., `Handler`, `Config`, `Service`)
- Struct fields: PascalCase if exported (e.g., `DatabaseURL`, `Port`, `Pool`)
- Struct field names align with JSON/form tags when applicable (e.g., `Username`, `PasswordHash`)

**Interfaces:**
- Reader-style naming: `Handler`, `Service` (concrete nouns)
- Explicit names for responsibility (e.g., `Service` for business logic, `Handler` for HTTP)

## Code Style

**Formatting:**
- Go standard formatting via `gofmt` (enforced by default)
- Template formatting: `templ fmt` (per Makefile)
- No custom formatting opinions beyond Go standard

**Linting:**
- Tool: golangci-lint with `.golangci.yml` config
- Key rules enabled:
  - `errcheck`: All errors must be checked
  - `govet`: Go vet checks
  - `ineffassign`: Unused variable assignments
  - `staticcheck`: Static analysis
  - `unused`: Unused variables/functions
- Excluded dirs: `internal/database/sqlc/` (generated code)
- Timeout: 5 minutes

**Code organization:**
- Import statements grouped and sorted by Go convention:
  1. Standard library packages
  2. External packages (third-party)
  3. Internal packages (project modules)
- One blank line between import groups
- Packages organized by responsibility (handler, auth, config, database, etc.)

## Import Organization

**Order:**
1. Standard library (e.g., `context`, `fmt`, `log/slog`, `net/http`)
2. Third-party packages (e.g., `github.com/labstack/echo/v4`, `github.com/jackc/pgx/v5`)
3. Internal packages (e.g., `docko/internal/auth`, `docko/internal/config`)

**Path Aliases:**
- No path aliases observed; all imports use full module path `docko/internal/...`
- Generated code prefixes: `docko/internal/database/sqlc` for SQLC queries

**Conventions:**
- No blank line before imports
- One blank line between import groups
- Imports kept minimal; only import what's used

## Error Handling

**Patterns:**
- Errors wrapped with context using `fmt.Errorf("message: %w", err)`
- Error messages describe what failed and why
- Example: `fmt.Errorf("failed to connect to database: %w", err)`
- Database errors checked explicitly: `if errors.Is(err, pgx.ErrNoRows)`
- HTTP errors handled via Echo context methods: `c.Redirect()`, `c.String()`
- Failed async operations logged at appropriate level (Warn for cleanup, Error for critical)

**Error propagation:**
- Functions return `error` as last return value
- Multiple returns formatted as: `(value, error)`
- Sentinel errors defined at package level (e.g., `var ErrInvalidCredentials = errors.New(...)`)
- Error checks immediately after function calls

**Example from `auth.go`:**
```go
if err != nil {
    return fmt.Errorf("failed to hash password: %w", err)
}
```

## Logging

**Framework:** `log/slog` (standard library logging)

**Setup:**
- Configured in `cmd/server/slog.go`
- Development: tint handler with colored output, time.Kitchen format
- Production: JSON handler with no colors
- Log level configurable via `LOG_LEVEL` env var (DEBUG, INFO, WARN, ERROR)
- Source file location added when `LOG_LEVEL=DEBUG`

**Patterns:**
- Use `slog.Info()` for normal operations
- Use `slog.Warn()` for unexpected but recoverable situations
- Use `slog.Error()` for failures that need attention
- Use `slog.Debug()` for detailed diagnostic info (development only)
- Pass structured fields as alternating key-value pairs

**Examples:**
```go
slog.Info("starting server", "url", "http://localhost:3000", "env", cfg.Env)
slog.Error("failed to connect to database", "error", err)
slog.Warn("failed login attempt", "username", username, "ip", c.RealIP())
```

## Comments

**When to Comment:**
- Public functions/methods: always include doc comment (not observed yet, opportunity)
- Non-obvious logic: explain why, not what
- Middleware: comment purpose and side effects
- Constants: explain the value's meaning if not obvious

**Style:**
- Doc comments start with the function name: `// NamedFunction does X`
- Inline comments start with `//` followed by space
- Avoid redundant comments (code should be self-documenting)

**Examples observed:**
```go
// RequireAuth middleware protects routes that require authentication
func RequireAuth(authService *auth.Service) echo.MiddlewareFunc {
```

## Function Design

**Size:** Aim for functions under 40 lines; break down complex logic

**Parameters:**
- Maximum 3-4 parameters; use structs for more
- Context as first parameter for all functions with I/O
- Handlers pass through `echo.Context` for request/response
- Services receive dependency-injected components (db, config)

**Return Values:**
- Error as last return value always
- Single meaningful value + error typical pattern: `(value, error)`
- No named returns (convention preference)

**Example pattern:**
```go
func (s *Service) ValidateCredentials(ctx context.Context, username, password string) (*sqlc.AdminUser, error) {
    // implementation
}
```

## Module Design

**Exports:**
- Exported names (PascalCase) for public API
- Unexported names (camelCase) for internal helpers
- Constructor functions named `New()` or `New<Type>()`
- No `Init()` functions; use constructors instead

**Barrel Files:**
- Not used in this codebase; each file exports its own public API
- No alias exports or re-exports observed

**Package structure:**
- One main struct per package (e.g., `Service` in auth, `Handler` in handler)
- Related functions attached as methods
- Helper functions unexported or in separate files

**Examples:**
```go
// handler/handler.go exports Handler
type Handler struct { ... }
func New(cfg *config.Config, db *database.DB, authService *auth.Service) *Handler { ... }

// auth/auth.go exports Service
type Service struct { ... }
func NewService(db *database.DB, cfg *config.Config) *Service { ... }
```

## Builder Pattern

**Meta construction:**
- `PageMeta` uses method chaining for optional fields
- Constructor: `New(title, description string) PageMeta`
- Chainable methods: `WithOGImage()`, `WithCanonical()`, `AsArticle()`, `AsProduct()`
- Returns modified struct copy (not pointer), enables clean chaining

**Example:**
```go
meta.New("Home", "Description").WithOGImage(url).WithCanonical(url)
```

## Context Keys

**Pattern:** Typed context keys to avoid string-based lookups

**Location:** `internal/ctxkeys/keys.go`

**Structure:**
```go
type adminUserKey struct{}
var AdminUser = adminUserKey{}  // used as context key

type siteConfigKey struct{}
var SiteConfig = siteConfigKey{}  // used as context key
```

**Usage:**
```go
ctx := context.WithValue(c.Request().Context(), ctxkeys.AdminUser, session.Username)
```

---

*Convention analysis: 2026-02-02*
