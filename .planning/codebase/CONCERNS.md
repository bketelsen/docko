# Codebase Concerns

**Analysis Date:** 2026-02-02

## Security Considerations

**CORS Configuration - Wildcard Origin:**
- Issue: CORS middleware allows `AllowOrigins: []string{"*"}` without restriction
- Files: `internal/middleware/middleware.go:20-22`
- Risk: Any origin can make cross-origin requests to the application. While origins are pre-flight checked, this is overly permissive for a web application with admin authentication
- Current mitigation: Only affects public endpoints; admin routes require authentication
- Recommendations: Restrict CORS to specific trusted origins. At minimum, validate `SITE_URL` env var and allow only that origin

**Content Security Policy - Inline Scripts:**
- Issue: CSP allows `'unsafe-inline'` for both scripts and styles
- Files: `internal/middleware/middleware.go:32`
- Risk: Reduces protection against XSS attacks. Any DOM-based XSS can execute code
- Current usage: Inline theme detection script in `templates/layouts/admin.templ:15-19` and theme toggle function in `templates/layouts/admin.templ:89-100`
- Recommendations: Move inline scripts to `assets/js/` and remove `'unsafe-inline'`. Use nonce-based CSP or move theme logic to external script with data attributes

**Session Secret Auto-generation in Development:**
- Issue: Session secret is randomly generated every server start in development
- Files: `internal/config/config.go:43`, `internal/config/config.go:84-88`
- Risk: Session tokens become invalid on server restart. Not ideal for development but acceptable
- Current mitigation: Only in development mode
- Recommendations: Allow explicit `SESSION_SECRET` env var to persist sessions across restarts for development convenience. Currently `SESSION_SECRET` env var is respected but auto-generation takes precedence when not set

**Default Credentials in Development:**
- Issue: `.envrc` contains default admin password `admin123`
- Files: `.envrc:18`
- Risk: If checked into version control or shared, compromises security of dev instance
- Current mitigation: `.envrc.example` exists as template; actual `.envrc` should not be committed
- Recommendations: Ensure `.envrc` is in `.gitignore` and never committed with real credentials

## Error Handling

**Silent Error Suppression in Auth Flow:**
- Issue: Session deletion errors are silently ignored in two places
- Files:
  - `internal/handler/auth.go:66` - Logout handler ignores DeleteSession error
  - `internal/auth/auth.go:79` - Password update ignores DeleteAdminUserSessions error
- Impact: If session deletion fails, user might not realize logout is incomplete (logout handler) or old sessions persist after password change
- Fix approach: Log the error even if it doesn't prevent the operation. E.g., `if err := h.auth.DeleteSession(...); err != nil { slog.Warn("failed to delete session on logout", "error", err) }`

**Rand.Read Error Ignored:**
- Issue: `rand.Read()` error is suppressed in `internal/config/config.go:86`
- Files: `internal/config/config.go:84-88`
- Impact: If random generation fails (extremely unlikely but possible), a weak secret is generated
- Severity: Low (cryptographic failures are rare), but poor practice
- Fix approach: Return error or panic: `if _, err := rand.Read(b); err != nil { panic(err) }`

## Data Flow Issues

**Login Redirect with Error Message as Query Parameter:**
- Issue: Error messages are passed as URL query parameters in redirects
- Files: `internal/handler/auth.go:32,38,44`
- Example: `/login?error=Please+enter+username+and+password`
- Risk: Error messages are logged in server logs and browser history. Doesn't expose sensitive data but not ideal
- Current mitigation: Messages are generic ("Invalid username or password" for all auth failures)
- Recommendations: Use flash messages with server-side storage instead of query params (requires session implementation for non-authenticated users)

**Query Parameter Not Validated/Escaped in Template:**
- Issue: `errorMsg := c.QueryParam("error")` is passed directly to template
- Files: `internal/handler/auth.go:22`, `templates/pages/admin/login.templ`
- Risk: If template doesn't escape, XSS vector exists (though templ auto-escapes by default)
- Current mitigation: Templ auto-escapes output
- Recommendations: Document that this is safe or remove and use flash messages

## Fragile Areas

**Authentication System - Single Admin User Model:**
- Files: `internal/auth/auth.go`, `internal/database/migrations/002_admin_users.sql`
- Why fragile:
  - Only supports one admin user (hardcoded `AdminUsername = "admin"`)
  - `ADMIN_PASSWORD` env var updates password on every startup, which invalidates all sessions
  - No user management features beyond password reset
- Safe modification: This design is intentional for single-admin apps. To extend: Create user management handler, modify schema to support multiple users, update SyncAdminUser logic
- Test coverage: No tests for auth service (no `*_test.go` files found)

**Session Validation Race Condition:**
- Issue: Session can be deleted or expire between validation check and usage
- Files: `internal/middleware/auth.go:24-29`
- Scenario: Cookie is validated as valid, but then: (1) session expires, (2) password is updated (invalidates session), or (3) user logs out from another tab
- Impact: Potential use of invalidated session tokens (though database queries will fail)
- Severity: Low to medium - subsequent database queries will fail, resulting in 500 error rather than graceful redirect
- Mitigation: None currently implemented
- Fix approach: Revalidate session on each request, or store session refresh_token separate from access_token (more complex)

**No Test Coverage:**
- Issue: No `_test.go` files found in codebase
- Files: All Go source files
- What's not tested:
  - Auth service (critical) - all password, session, and credential validation logic
  - Middleware (critical) - RequireAuth middleware and CORS/security middleware
  - Handler (important) - login, logout, dashboard handlers
  - Database connectivity (important) - connection pooling, migrations
- Risk: Regressions and bugs can slip into auth flow, which is security-critical
- Priority: High
- Recommendations: Add test suite covering at minimum:
  1. Auth service: ValidateCredentials, CreateSession, ValidateSession, SyncAdminUser
  2. Middleware: RequireAuth with valid/invalid/expired tokens
  3. Handlers: Login with valid/invalid credentials, logout, redirect to login when unauthorized
  4. Integration tests with test database using `internal/testutil`

## Scaling Limits

**Session Cleanup Runs on Single Server Instance:**
- Issue: Expired session cleanup is scheduled in-process on a 1-hour interval
- Files: `cmd/server/main.go:39-47`
- Current capacity: Single goroutine, runs once per hour. Acceptable for small deployments
- Limit: In multi-instance deployments, each instance runs cleanup independently, causing redundant queries
- Scaling path:
  1. For multiple instances: Use job scheduling service (e.g., celery, bull, temporal) or implement distributed cleanup with leader election
  2. Or: Move cleanup to PostgreSQL trigger: `CREATE TRIGGER cleanup_expired_sessions AFTER UPDATE ON admin_sessions ...`

**Static File Serving from Working Directory:**
- Issue: `e.Static("/static", "static")` and `e.Static("/assets", "assets")` serve files relative to current working directory
- Files: `internal/handler/handler.go:28-29`
- Problem: If app is run from wrong directory or deployed without static files, requests will 404 silently
- Recommendations:
  1. Embed static files using `//go:embed` (best for distribution)
  2. Or validate static directory exists on startup
  3. Or serve from absolute path

## Performance Bottlenecks

**Session Token Hashing on Every Validation:**
- Issue: Session token is hashed using HMAC-SHA256 on every request validation
- Files: `internal/auth/auth.go:142-146` called from `internal/middleware/auth.go:24`
- Cause: Token is extracted from cookie and hashed before database lookup
- Impact: ~1-2ms per request (minimal impact but unnecessary)
- Improvement path:
  1. Current design is actually good for security (makes token compromise less valuable since they're hashed in DB)
  2. If needed: Cache the hash result per request using context
  3. Or: Use stateless JWTs instead (requires redesign)

**No Database Connection Pooling Configuration:**
- Issue: `pgxpool.New()` uses default pool size (4 connections)
- Files: `internal/database/database.go:24`
- Cause: No explicit `MaxConns` configuration
- Impact: Under high load (>4 concurrent requests with long-running queries), connection pool exhaustion
- Limit: Default 4 connections is suitable for small apps but will bottleneck at moderate scale
- Improvement path:
  1. Add `DB_MAX_CONNECTIONS` env var with default 25
  2. Configure pool in `database.New()`: `config := pgxpool.ParseConfig(...); config.MaxConns = maxConns; pool, _ := pgxpool.NewWithConfig(...)`

## Dependencies at Risk

**templ Transitive Dependency on Tailwind Merge:**
- Package: `github.com/Oudwins/tailwind-merge-go v0.2.1`
- Risk: Small project with single maintainer. May have limited community support
- Impact: If dependency breaks or stops being maintained, will need to fork or rewrite
- Migration plan: Dependency is only used by templUI components. Could migrate components to use plain Tailwind classnames without merging (removes value though)

**Transitive Security Dependencies:**
- Packages: `golang.org/x/crypto`, `golang.org/x/net`, `golang.org/x/sys`
- Risk: These are actively maintained but critical for security (bcrypt, TLS)
- Current version: v0.40.0 for crypto, v0.42.0 for net
- Recommendations:
  1. Add GitHub dependabot configuration
  2. Regularly run `go get -u` in CI
  3. Monitor security advisories for Go

## Missing Critical Features

**No Rate Limiting on Login:**
- Problem: `/login` endpoint has no rate limiting
- Blocks: Brute force attacks are possible
- Files: `internal/handler/auth.go:27-60`
- Recommendations: Implement rate limiting middleware
  1. Per IP: Allow 5 failed login attempts per 15 minutes
  2. Use middleware like `github.com/labstack/echo-contrib/middleware/ratelimit`
  3. Or implement simple in-memory tracking with cleanup

**No CSRF Protection:**
- Problem: Form submissions (login, logout) lack CSRF tokens
- Blocks: CSRF attacks on authenticated users
- Files: `internal/handler/auth.go` (login form), `templates/pages/admin/login.templ` (logout button in admin.templ)
- Risk: Low for password-based login (requires user to be tricked at login time), High for logout (user can be logged out)
- Recommendations:
  1. Add CSRF token middleware (e.g., `github.com/labstack/echo-contrib/middleware/csrf`)
  2. Include CSRF token in forms and validate on POST
  3. Or: Use SameSite=Strict cookies (partially addresses, but not sufficient)

**No Request Logging for Admin Actions:**
- Problem: Dashboard page (`admin.go:AdminDashboard`) does nothing - no audit trail
- Blocks: Cannot track admin activities for compliance/debugging
- Files: `internal/handler/admin.go:11-13`
- Recommendations:
  1. Add request ID to logs (already done by middleware)
  2. Add audit table: `CREATE TABLE audit_logs (id UUID, user_id UUID, action TEXT, timestamp TIMESTAMPTZ, ...)`
  3. Log admin actions to audit table

**No Password Complexity Validation:**
- Problem: `ADMIN_PASSWORD` env var is accepted as-is with no complexity rules
- Blocks: Weak passwords accepted
- Files: `internal/config/config.go:42`, `internal/auth/auth.go:46`
- Recommendations:
  1. Add password complexity function (minimum 12 chars, upper/lower/number/symbol)
  2. Validate in `config.Load()` or `auth.SyncAdminUser()`
  3. Or: Accept minimal constraint and document in `.envrc.example`

## Test Coverage Gaps

**Critical Auth System Untested:**
- What's not tested: Session creation, validation, expiration, password hashing, credential validation
- Files: `internal/auth/auth.go` (all)
- Risk: Auth logic can regress without breaking other code
- Priority: High
- Suggested tests:
  ```go
  // Test ValidateCredentials with valid/invalid password
  // Test CreateSession generates valid token
  // Test ValidateSession rejects expired sessions
  // Test ValidateSession rejects tampered tokens
  // Test SyncAdminUser creates new user
  // Test SyncAdminUser updates password and invalidates sessions
  ```

**Middleware Authorization Not Tested:**
- What's not tested: RequireAuth with valid/invalid/expired tokens, cookie handling
- Files: `internal/middleware/auth.go:16-38`
- Risk: Authorization bypass not caught
- Priority: High
- Suggested tests:
  ```go
  // Test RequireAuth with no cookie → redirect to /login
  // Test RequireAuth with invalid token → redirect to /login
  // Test RequireAuth with expired token → redirect to /login, clear cookie
  // Test RequireAuth with valid token → allow access, set context
  ```

**Handler Logic Minimally Tested:**
- What's not tested: Login with valid/invalid credentials, logout flow, session cookie setting
- Files: `internal/handler/auth.go` (all)
- Risk: Login/logout flow can break silently
- Priority: Medium
- Suggested tests:
  ```go
  // Test LoginPage redirects to dashboard if already authenticated
  // Test Login with empty credentials → error redirect
  // Test Login with invalid credentials → error redirect, log warning
  // Test Login with valid credentials → sets cookie, redirects to /
  // Test Logout → clears session and cookie
  ```

---

*Concerns audit: 2026-02-02*
