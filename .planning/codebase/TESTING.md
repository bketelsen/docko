# Testing Patterns

**Analysis Date:** 2026-02-02

## Test Framework

**Runner:**
- Go built-in `go test`
- Configuration: None required (uses Go defaults)
- Race detection enabled by default in Makefile

**Assertion Library:**
- None; standard Go testing practices use `if` statements and `t.Fatalf()`

**Run Commands:**
```bash
make test              # Run all tests with race detection (go test -v -race ./...)
go test ./...          # Run all tests verbose
go test -cover ./...   # Run with coverage report
```

**Test discovery:**
- Files matching `*_test.go` pattern
- Functions matching `Test*` prefix

## Test File Organization

**Location:**
- Co-located with implementation files (same package, same directory)
- Not in separate test directory

**Naming:**
- `<subject>_test.go` for test files
- `Test<FunctionName>` for test functions
- Example: `auth_test.go` contains tests for `auth.go`

**Structure:**
- Tests in same package (e.g., package `auth` for `auth_test.go`)
- Allows testing unexported functions
- Tests co-located with code for easy modification during development

## Test Helpers

**Location:** `internal/testutil/testutil.go`

**Available helpers:**

### Database test setup:
```go
// NewTestDB creates a test database connection.
// Requires TEST_DATABASE_URL environment variable or uses DATABASE_URL.
func NewTestDB(t *testing.T) *database.DB {
    t.Helper()

    ctx := context.Background()
    dbURL := os.Getenv("TEST_DATABASE_URL")
    if dbURL == "" {
        dbURL = os.Getenv("DATABASE_URL")
    }
    if dbURL == "" {
        t.Skip("TEST_DATABASE_URL or DATABASE_URL not set, skipping database test")
    }

    db, err := database.New(ctx, dbURL)
    if err != nil {
        t.Fatalf("failed to create test database: %v", err)
    }

    t.Cleanup(func() {
        db.Close()
    })

    return db
}
```

### Configuration test setup:
```go
// NewTestConfig creates a test configuration.
func NewTestConfig(t *testing.T) *config.Config {
    t.Helper()

    return &config.Config{
        DatabaseURL: os.Getenv("TEST_DATABASE_URL"),
        Port:        "0",
        Env:         "test",
        Site: config.SiteConfig{
            Name: "docko",
            URL:  "http://localhost:3000",
        },
    }
}
```

**Key patterns:**
- All helpers call `t.Helper()` to report line numbers in test files, not helpers
- Database tests skip if environment variables missing (no test DB required to run tests)
- Config uses Port "0" (OS assigns random port for testing)
- Env set to "test" for test-specific behavior
- `t.Cleanup()` ensures resources are freed after test completes

## Test Structure

**Standard Go testing structure:**

```go
func TestFunctionName(t *testing.T) {
    // Setup

    // Execute

    // Assert
}
```

**Pattern example (not yet implemented, but recommended):**
```go
func TestValidateCredentials_InvalidPassword(t *testing.T) {
    // Setup
    db := testutil.NewTestDB(t)
    cfg := testutil.NewTestConfig(t)
    authService := auth.NewService(db, cfg)
    ctx := context.Background()

    // Execute
    user, err := authService.ValidateCredentials(ctx, "admin", "wrongpassword")

    // Assert
    if err == nil {
        t.Fatal("expected error for invalid password")
    }
    if user != nil {
        t.Fatal("expected user to be nil on auth failure")
    }
}
```

## Environment Setup for Tests

**Required environment variables:**
- `TEST_DATABASE_URL` or `DATABASE_URL`: PostgreSQL connection string
- If neither is set, tests skip gracefully

**Example for development:**
```bash
# Load direnv config (includes DATABASE_URL for docker-compose)
direnv allow

# Run tests
make test
```

**Example for CI:**
```bash
# Set TEST_DATABASE_URL pointing to test database
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/test_docko"
make test
```

## Mocking

**Current state:** No mocking framework detected (no imports of testify, gomock, etc.)

**Approach when mocking is needed:**
- Go standard library `*testing.T` and manual setup
- Consider interfaces for dependency injection
- Example pattern (not yet used):
```go
type MockAuthService struct {
    ValidateCredentialsFunc func(ctx context.Context, username, password string) (*User, error)
}

func (m *MockAuthService) ValidateCredentials(ctx context.Context, username, password string) (*User, error) {
    return m.ValidateCredentialsFunc(ctx, username, password)
}
```

**Strategy:**
- Keep dependencies injectable (all services use constructor injection)
- Use interfaces for integration points (auth.Service, database.DB)
- Avoid mocking database layer; use testutil.NewTestDB() instead
- Mock external APIs via interface{} parameters if needed

## Table-Driven Tests

**Pattern recommendation (not yet implemented):**
```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        hash     string
        wantErr  bool
    }{
        {"valid password", "password123", hash, false},
        {"invalid password", "wrong", hash, true},
        {"empty password", "", hash, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := bcrypt.CompareHashAndPassword([]byte(tt.hash), []byte(tt.password))
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, want error %v", err != nil, tt.wantErr)
            }
        })
    }
}
```

## Coverage

**Requirements:** None enforced (no .codecov.yml or CI checks)

**View Coverage:**
```bash
go test -cover ./...                  # Summary of all packages
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out      # HTML report
```

**Areas with test utilities but no tests written yet:**
- `internal/testutil/` provides helpers but no test files found
- Database integration tests not yet created
- Auth service tests not yet created
- Handler tests not yet created

## Test Types

**Unit Tests:**
- Scope: Test individual functions/methods in isolation
- Approach: Direct function calls with simple inputs, verify outputs
- Use: Test business logic (auth validation, config loading, etc.)
- Setup: Use testutil helpers for dependencies

**Integration Tests:**
- Scope: Test multiple components working together
- Approach: Use real database (test instance via TEST_DATABASE_URL)
- Use: Test full request/response flows, database operations
- Setup: `testutil.NewTestDB()` sets up schema via migrations

**E2E Tests:**
- Framework: Not used
- Alternative: Use integration tests with HTTP client for handler testing

## Async Testing

**Context handling:**
```go
ctx := context.Background()  // For tests that don't need timeouts
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

**Pattern (when implemented):**
```go
func TestAsyncOperation(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    done := make(chan error)
    go func() {
        done <- longRunningOperation(ctx)
    }()

    select {
    case err := <-done:
        if err != nil {
            t.Fatalf("async operation failed: %v", err)
        }
    case <-ctx.Done():
        t.Fatal("operation timed out")
    }
}
```

## Error Testing

**Pattern (when implemented):**
```go
func TestValidateCredentials_InvalidUser(t *testing.T) {
    db := testutil.NewTestDB(t)
    authService := auth.NewService(db, testutil.NewTestConfig(t))
    ctx := context.Background()

    _, err := authService.ValidateCredentials(ctx, "nonexistent", "password")

    if err == nil {
        t.Fatal("expected error for nonexistent user")
    }

    if !errors.Is(err, auth.ErrInvalidCredentials) {
        t.Fatalf("expected ErrInvalidCredentials, got %v", err)
    }
}
```

**Sentinel error testing:**
- Use `errors.Is()` for custom error types
- Define error variables at package level
- Test both error occurrence and error type

## Performance Testing

**Not implemented:**

Pattern recommendation:
```bash
go test -bench=. -benchmem ./...     # Run benchmarks with memory stats
```

**When to add:**
- Password hashing performance (bcrypt is intentionally slow)
- Database query performance under load
- Session token generation speed

## Best Practices Applied

**What's done well:**
- Helper functions use `t.Helper()` for better error reporting
- Database setup skips gracefully when test DB unavailable
- Config provides test-safe defaults (Port "0", Env "test")
- All errors wrapped with context

**What's not yet done:**
- No actual test files written (`*_test.go`)
- No test coverage measurement
- No integration tests using testutil helpers
- No mocking patterns established

## Adding Tests

**Checklist for writing new tests:**
1. Create `<module>_test.go` in same package as code
2. Import `testing` and required dependencies
3. Use `testutil.NewTestDB()` for database tests
4. Use `testutil.NewTestConfig()` for config-dependent code
5. Call `t.Helper()` in any helper functions
6. Use `t.Cleanup()` for resource cleanup
7. Run `make test` to verify
8. Check error messages are descriptive in assertions

**Template for test file:**
```go
package auth

import (
    "context"
    "testing"
    "docko/internal/testutil"
)

func TestFunctionName(t *testing.T) {
    db := testutil.NewTestDB(t)
    cfg := testutil.NewTestConfig(t)
    ctx := context.Background()

    // Arrange

    // Act

    // Assert
}
```

---

*Testing analysis: 2026-02-02*
