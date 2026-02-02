# Codebase Structure

**Analysis Date:** 2026-02-02

## Directory Layout

```
docko/
├── cmd/server/              # Application entry point
│   ├── main.go              # Server initialization, routing setup
│   ├── generate.go          # Go generate directives for code generation
│   └── slog.go              # Logging configuration
├── internal/                # Private application code
│   ├── auth/                # Authentication & session management
│   ├── config/              # Environment configuration
│   ├── ctxkeys/             # Typed context keys
│   ├── database/            # Database connection & migrations
│   │   ├── migrations/      # Goose SQL migration files
│   │   └── sqlc/            # sqlc-generated query code
│   ├── handler/             # HTTP request handlers
│   ├── meta/                # SEO metadata helpers
│   ├── middleware/          # Echo middleware (auth, logging, security)
│   └── testutil/            # Test utilities
├── templates/               # Templ template files
│   ├── layouts/             # Base layouts (base.templ, admin.templ, login.templ)
│   └── pages/               # Page templates
│       └── admin/           # Admin-specific pages
├── components/              # templUI components
│   ├── button/
│   ├── card/
│   ├── input/
│   ├── label/
│   └── [other components]/
├── assets/                  # Client-side assets
│   └── js/                  # JavaScript libraries (templUI assets)
├── static/                  # Static files served to browser
│   ├── css/                 # Tailwind input.css and compiled output.css
│   ├── js/                  # Custom JavaScript files
│   └── images/              # Static images
├── sqlc/                    # sqlc configuration and queries
│   ├── sqlc.yaml            # sqlc configuration file
│   └── queries/             # SQL query files for code generation
├── utils/                   # Utility functions
│   └── templui.go           # templUI Tailwind merge and helper functions
├── go.mod                   # Go module definition
├── go.sum                   # Go dependency checksums
├── Makefile                 # Build and development commands
├── docker-compose.yml       # PostgreSQL for development
├── .envrc                   # direnv environment configuration
└── .planning/               # Project planning documents
    └── codebase/            # Codebase analysis documents
```

## Directory Purposes

**cmd/server:**
- Purpose: Binary entry point and initialization logic
- Contains: Server startup, middleware setup, go:generate directives
- Key files: `main.go` initializes config, database, auth, creates Echo instance

**internal/auth:**
- Purpose: Authentication service and session management
- Contains: Password hashing (bcrypt), token generation, session validation, admin user sync
- Key files: `auth.go` with Service methods for credentials validation and session CRUD

**internal/config:**
- Purpose: Environment variable loading and configuration struct
- Contains: Config struct with nested SiteConfig and AuthConfig, helper functions
- Key files: `config.go` loads env vars with defaults, provides IsProduction() helper

**internal/ctxkeys:**
- Purpose: Typed context key definitions
- Contains: Private struct types for context keys (prevents key collisions)
- Key files: `keys.go` exports SiteConfig and AdminUser key variables

**internal/database:**
- Purpose: Database connection, pooling, migrations, and query interface
- Contains: pgx connection pool, goose migrations, sqlc query wrappers
- Key files: `database.go` manages pgxpool, runs migrations on startup
- Subdir `migrations/`: SQL files for goose (001_initial.sql, 002_admin_users.sql)
- Subdir `sqlc/`: Generated Go code from sqlc (queries, models)

**internal/handler:**
- Purpose: HTTP request handlers for all routes
- Contains: Handler struct (with cfg, db, auth dependencies), route registration, form processing
- Key files: `handler.go` (RegisterRoutes, Handler struct), `auth.go` (login/logout), `admin.go` (dashboard)

**internal/meta:**
- Purpose: SEO/OG metadata for templates
- Contains: PageMeta struct, builder methods, context helpers to extract site config
- Key files: `meta.go` (PageMeta), `context.go` (helper functions to read from context)

**internal/middleware:**
- Purpose: Echo middleware for cross-cutting concerns
- Contains: Auth protection, request logging, CORS, security headers, site config injection
- Key files: `middleware.go` (Setup function, request logger), `auth.go` (RequireAuth middleware)

**internal/testutil:**
- Purpose: Helpers and utilities for testing
- Contains: Test database setup, mock factories, assertion helpers
- Key files: Depends on project's testing needs

**templates:**
- Purpose: Templ template files compiled to Go
- Contains: HTML structure, dynamic content, component composition
- Key files: `layouts/base.templ` (public layout), `layouts/admin.templ` (admin with sidebar), `layouts/login.templ` (login page)
- Subdir `pages/admin/`: `login.templ`, `dashboard.templ`

**components:**
- Purpose: Reusable UI components from templUI
- Contains: templUI component templates (button, card, input, label, etc.)
- Key files: Each component has `.templ` source and `_templ.go` generated code

**assets:**
- Purpose: Client-side asset files (JavaScript libraries)
- Contains: templUI JavaScript files needed by components
- Key files: `js/` subdirectory with templUI JS libraries

**static:**
- Purpose: Assets served directly to browser (not processed)
- Contains: CSS (Tailwind compiled), custom JS, images
- Key files: `css/input.css` (Tailwind directives), `css/output.css` (compiled)

**sqlc:**
- Purpose: sqlc configuration and SQL query definitions
- Contains: Configuration for code generation, parameterized SQL queries
- Key files: `sqlc.yaml` (config pointing to schema, queries, output), `queries/admin_auth.sql`, `queries/example.sql`

**utils:**
- Purpose: General utility functions
- Contains: Tailwind class merging (tailwind-merge-go), templUI helpers
- Key files: `templui.go` with TwMerge, If, IfElse, MergeAttributes, RandomID, cache busting functions

## Key File Locations

**Entry Points:**
- `cmd/server/main.go`: Creates config, database, auth service, Echo instance, registers routes, starts server
- `cmd/server/slog.go`: Initializes structured logging at package init time
- `internal/handler/handler.go`: RegisterRoutes() defines all HTTP routes and their handlers

**Configuration:**
- `internal/config/config.go`: Loads all environment variables, provides Config struct
- `.envrc`: direnv file with default environment variables for development
- `docker-compose.yml`: PostgreSQL service definition for local development

**Core Logic:**
- `internal/auth/auth.go`: Session management, password hashing, admin user sync
- `internal/handler/auth.go`: Login/logout form handling
- `internal/handler/admin.go`: Dashboard rendering
- `internal/database/database.go`: Connection pooling and migration runner

**Database:**
- `internal/database/migrations/`: SQL migration files (goose format)
- `sqlc/queries/`: SQL query definitions for code generation
- `sqlc/sqlc.yaml`: Configuration telling sqlc where schema and queries are
- `internal/database/sqlc/`: Generated Go code (not to be edited manually)

**Templates:**
- `templates/layouts/base.templ`: Public site layout with header/footer
- `templates/layouts/admin.templ`: Admin dashboard layout with sidebar and theme toggle
- `templates/layouts/login.templ`: Login page layout
- `templates/pages/admin/login.templ`: Login form using templUI components
- `templates/pages/admin/dashboard.templ`: Dashboard with stat cards and quick actions

**Security:**
- `internal/middleware/auth.go`: RequireAuth() middleware protecting routes
- `internal/auth/auth.go`: Password hashing and token validation logic
- `internal/database/migrations/002_admin_users.sql`: admin_users and admin_sessions tables

## Naming Conventions

**Files:**
- Go source: snake_case.go (e.g., `admin.go`, `auth.go`, `user_service.go`)
- Templates: snake_case.templ (e.g., `admin.templ`, `login.templ`)
- SQL migrations: `001_description.sql` (numbered prefix with goose format)
- SQL queries: snake_case.sql (e.g., `admin_auth.sql`, `user_queries.sql`)
- Generated Go from templates/queries: `filename_templ.go`, `filename.sql.go` (generated by tools)

**Directories:**
- Internal packages: lowercase, no underscores (e.g., `auth`, `handler`, `middleware`)
- Template subdirs: lowercase plural or descriptive (e.g., `layouts`, `pages`)
- Component dirs: component name in lowercase (e.g., `button`, `card`, `input`)

**Go Functions/Methods:**
- Exported (public): PascalCase (e.g., `New()`, `RegisterRoutes()`, `ValidateCredentials()`)
- Unexported (private): camelCase (e.g., `migrate()`, `hashToken()`)
- Interface methods: PascalCase following Go conventions

**Go Types:**
- Structs: PascalCase (e.g., `Handler`, `Config`, `PageMeta`, `Service`)
- Interfaces: PascalCase (e.g., `HandlerFunc`)

**Variables:**
- Context keys: PascalCase exported from ctxkeys package (e.g., `ctxkeys.AdminUser`)
- Constants: UPPER_SNAKE_CASE (e.g., `SessionCookieName`, `TokenLength`, `AdminUsername`)

## Where to Add New Code

**New Route Handler:**
- Determine if it's auth-related (goes in `internal/handler/auth.go`) or admin (goes in `internal/handler/admin.go`)
- Add route to `RegisterRoutes()` in `internal/handler/handler.go`
- Create template in `templates/pages/[feature]/` if needed
- Example: POST /api/users would go in new `internal/handler/api.go` if api endpoints needed

**New Template Page:**
- Create `.templ` file in `templates/pages/[feature]/` with package name matching directory
- Use `@layouts.Base()` or `@layouts.Admin()` depending on protected/public
- Construct `PageMeta` inside the template: `meta.New("Title", "Description")`
- Import and use templUI components: `@button.Button()`, `@card.Card()`

**New Database Table/Query:**
- Create `.sql` migration file in `internal/database/migrations/` with next number (e.g., `003_feature.sql`)
- Add query definitions in `sqlc/queries/feature.sql`
- Run `make generate` to create Go code in `internal/database/sqlc/`
- Call generated query: `h.db.Queries.QueryName(ctx, params)`

**New Service/Business Logic:**
- Create `internal/[feature]/service.go` (or similar)
- Inject dependencies through `NewService(cfg, db, ...)` constructor
- Pass service instance to Handler in `cmd/server/main.go`
- Call service methods from handler

**New templUI Component:**
- Run `templui add component-name` (one-time CLI)
- Customizations go in `components/[component-name]/[component-name].templ`
- Use in templates: `@componentname.Component(props)`

**New Utility Function:**
- Go utils: Add to `utils/` with appropriate file name
- Template utilities: Add to component `.templ` files as templ functions
- Middleware utilities: Add to `internal/middleware/middleware.go`

## Special Directories

**internal/database/migrations:**
- Purpose: Goose SQL migration files
- Generated: No, manually written
- Committed: Yes
- Format: `NNN_description.sql` with `-- +goose Up/Down` markers
- Auto-runs: On startup in `database.migrate()`

**internal/database/sqlc:**
- Purpose: Generated Go code from sqlc
- Generated: Yes, by sqlc from `sqlc/queries/` and `internal/database/migrations/`
- Committed: Yes (checked in after generation)
- Manual editing: Never - regenerate with `make generate` or `sqlc generate -f sqlc/sqlc.yaml`

**templates/layouts/:, templates/pages/:**
- Purpose: Templ source files
- Generated: No, hand-written
- Committed: Yes
- Auto-generated side effect: `*_templ.go` files created by `templ generate`

**components/:**
- Purpose: Templ component definitions (mostly from templUI)
- Generated: Installed by `templui add`, but source `.templ` files are editable
- Committed: Yes for customizations
- Auto-generated side effect: `*_templ.go` created by `templ generate`

**static/css/:**
- Purpose: Tailwind CSS
- Input: `input.css` (Tailwind directives and custom CSS)
- Output: `output.css` (compiled, used in templates)
- Generated: `output.css` by Tailwind CLI during `make generate`
- Committed: Both input.css and output.css committed

**tmp/:**
- Purpose: Development temporary files
- Contains: Hot-reload server logs, build artifacts
- Generated: Yes, by development tools
- Committed: No (in .gitignore)

**assets/js/:**
- Purpose: Client-side JavaScript assets (templUI library files)
- Generated: Installed by `templui add`
- Committed: Yes
- Manual editing: Not typically needed

---

*Structure analysis: 2026-02-02*
