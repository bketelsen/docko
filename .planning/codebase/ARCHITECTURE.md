# Architecture

**Analysis Date:** 2026-02-02

## Pattern Overview

**Overall:** Layered MVC-inspired web application with clear separation of concerns.

**Key Characteristics:**
- Request/response handled through Echo web framework
- Business logic encapsulated in service layers (auth, database)
- Template rendering with Templ (type-safe HTML generation)
- Database operations abstracted through sqlc-generated query functions
- Configuration via environment variables loaded at startup
- Middleware for cross-cutting concerns (logging, auth, security)

## Layers

**Transport/Handler Layer:**
- Purpose: HTTP request handling and routing
- Location: `internal/handler/`
- Contains: Handler structs, route registration, form processing
- Depends on: Auth service, Database, Config
- Used by: Echo framework

**Service Layer:**
- Purpose: Business logic and data operations
- Location: `internal/auth/`
- Contains: Authentication logic, session management, password hashing
- Depends on: Database, Config
- Used by: Handler layer, Middleware

**Database Layer:**
- Purpose: Data persistence and query execution
- Location: `internal/database/`
- Contains: Connection pooling, migrations, sqlc query wrappers
- Depends on: PostgreSQL, sqlc-generated code
- Used by: Auth service, Handler layer

**Presentation Layer:**
- Purpose: HTML template rendering
- Location: `templates/`, `components/`
- Contains: Templ templates, layout compositions, UI components
- Depends on: Meta (SEO helpers), templUI components
- Used by: Handler layer (via Render())

**Middleware Layer:**
- Purpose: Cross-cutting HTTP concerns
- Location: `internal/middleware/`
- Contains: Auth protection, logging, CORS, security headers
- Depends on: Auth service, Config
- Used by: Echo (applies to all routes or specific ones)

**Configuration Layer:**
- Purpose: Environment and application settings
- Location: `internal/config/`
- Contains: Config struct, environment variable loading
- Depends on: OS environment
- Used by: Main, all other layers

## Data Flow

**Authenticated Request Flow:**

1. HTTP request arrives → Echo routing
2. `RequireAuth` middleware intercepts
3. Extracts `admin_session` cookie from request
4. Calls `authService.ValidateSession()` to check token hash against database
5. If valid, adds user info to request context via `ctxkeys.AdminUser`
6. Handler receives request with authenticated context
7. Handler loads template and renders response
8. Response written to client

**Login Flow:**

1. User submits `/login` form with username/password
2. `Handler.Login()` receives form values
3. Calls `authService.ValidateCredentials(username, password)`
4. Service queries database for user by username
5. Compares submitted password against bcrypt hash
6. If valid, calls `authService.CreateSession(userID)`
7. Session service generates random token, hashes it with HMAC-SHA256, stores hash in database
8. Returns unhashed token to handler
9. Handler sets `admin_session` cookie (HttpOnly, Secure, SameSite=Lax)
10. Redirects to `/` (protected by `RequireAuth`)

**Page Rendering Flow:**

1. Handler function receives Echo context
2. Calls template function: `pages.Home().Render(ctx, responseWriter)`
3. Template function constructs `PageMeta` with title/description/OG tags
4. Passes meta to layout template: `@layouts.Base(meta)`
5. Layout accesses site config from context: `meta.SiteNameFromCtx(ctx)`
6. Renders HTML with dynamic values from context
7. Templ generates the final HTML to response writer

**State Management:**
- No global state except config and logger (initialized at startup)
- Session state stored in PostgreSQL `admin_sessions` table
- User state passed via typed context keys (`ctxkeys.AdminUser`, `ctxkeys.SiteConfig`)
- Theme preference persisted to browser localStorage (client-side)

## Key Abstractions

**Handler Struct:**
- Purpose: Bundles HTTP handler methods with dependencies (config, db, auth)
- Examples: `internal/handler/handler.go`, `internal/handler/auth.go`, `internal/handler/admin.go`
- Pattern: Receiver methods on Handler struct with signature `(h *Handler) MethodName(c echo.Context) error`

**Auth Service:**
- Purpose: Encapsulates authentication and session management logic
- Examples: `internal/auth/auth.go`
- Pattern: Methods for credential validation, session creation/validation, cleanup

**Database Wrapper:**
- Purpose: Abstracts connection pooling and migrations
- Examples: `internal/database/database.go`
- Pattern: DB struct wrapping pgxpool.Pool and sqlc.Queries, migrations run automatically on New()

**PageMeta:**
- Purpose: SEO/OG metadata for templates without handler coupling
- Examples: `internal/meta/meta.go`
- Pattern: Struct with builder methods (fluent interface): `meta.New().WithOGImage().AsArticle()`

**Config:**
- Purpose: All environment-driven configuration in one place
- Examples: `internal/config/config.go`
- Pattern: Loaded once at startup, immutable after initialization, passed as dependency

**Typed Context Keys:**
- Purpose: Type-safe context value storage without magic strings
- Examples: `internal/ctxkeys/keys.go`
- Pattern: Private struct types as keys, exported var instances (e.g., `ctxkeys.AdminUser`)

## Entry Points

**Server Startup:**
- Location: `cmd/server/main.go`
- Triggers: `go run cmd/server/main.go` or binary execution
- Responsibilities: Load config, connect database, initialize auth, register middleware, start Echo server, handle graceful shutdown

**Request Routing:**
- Location: `internal/handler/handler.go` - `RegisterRoutes()` method
- Triggers: Called during server startup
- Responsibilities: Define all route paths, associate handlers, apply route-specific middleware

**Template Rendering:**
- Location: All `templ` files in `templates/pages/` and `templates/layouts/`
- Triggers: Handler calls `.Render(ctx, responseWriter)`
- Responsibilities: Convert data to HTML, apply layouts, render components

## Error Handling

**Strategy:** Explicit error returns with context wrapping.

**Patterns:**
- Service functions return `error` as second return value
- Errors wrapped with `fmt.Errorf("context: %w", err)` to maintain error chain
- Handlers catch errors from services/database and respond with redirects (auth flow) or HTTP status (health check)
- Database constraint violations (e.g., unique username) result in errors that propagate to handler
- Invalid session/token returns `fmt.Errorf("invalid session")` to distinguish from other errors
- Auth middleware on invalid session clears cookie and redirects to `/login` (no error page)

## Cross-Cutting Concerns

**Logging:** `slog` library (Go stdlib) with environment-aware handlers. Development mode uses colored `tint` format with timestamps. Production uses JSON. Log level configurable via `LOG_LEVEL` env var. Request logging captures method, URI, status, latency.

**Validation:** Form values extracted directly from request in handlers. No centralized validator. Password validation delegated to bcrypt.CompareHashAndPassword(). Username/password presence checked in handler before auth service call.

**Authentication:** Session-cookie based. Raw token returned to client in cookie, hashed token stored in database. Token validation checks hash match and expiry against current time. Admin user synced from `ADMIN_PASSWORD` env var at startup using bcrypt hashing.

**Security Headers:** Applied globally by `middleware.Secure()` with CSP allowing self, htmx from unpkg, inline styles/scripts. XFrame: SAMEORIGIN, HSTS: 31536000 seconds. HttpOnly and Secure cookies in production.

---

*Architecture analysis: 2026-02-02*
