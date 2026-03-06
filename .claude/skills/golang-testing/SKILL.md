---
name: golang-testing
description: >
  Go testing standards: table-driven tests, testify, mockgen, coverage measurement,
  race-safe tests, benchmark correctness, and test isolation patterns.
  Use when writing, reviewing, or running Go tests.
allowed-tools: Bash(go test *), Bash(mockgen *), Bash(go generate *), Bash(go tool cover *), Bash(gotestsum *), Read, Grep, Glob
---

# Go Testing Standards

Tests in Go are first-class code. They live next to the implementation, run fast via `go test`,
and the race detector is always available. This skill enforces patterns that make tests
readable, reliable, and maintainable.

---

## 1. File & Package Naming

```
mypackage/
├── service.go           # production code — package mypackage
├── service_test.go      # white-box tests — package mypackage
└── service_ext_test.go  # black-box tests — package mypackage_test
```

**Convention:** one `_test.go` file per source file. Black-box tests (`_test` package suffix)
test the public API and are preferred for packages with stable interfaces.

---

## 2. Test Naming

```go
// Pattern: TestFunctionName_Scenario_ExpectedResult
func TestGetUser_ValidID_ReturnsUser(t *testing.T) { ... }
func TestGetUser_NotFound_ReturnsErrNotFound(t *testing.T) { ... }
func TestGetUser_DBError_WrapsError(t *testing.T) { ... }

// Subtests pattern:
func TestGetUser(t *testing.T) {
    t.Run("valid ID returns user", func(t *testing.T) { ... })
    t.Run("not found returns ErrNotFound", func(t *testing.T) { ... })
    t.Run("DB error is wrapped", func(t *testing.T) { ... })
}
```

---

## 3. Table-Driven Tests

The standard Go pattern for testing multiple scenarios:

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {name: "positive numbers", a: 1, b: 2, expected: 3},
        {name: "negative numbers", a: -1, b: -2, expected: -3},
        {name: "zero", a: 0, b: 0, expected: 0},
    }

    for _, tc := range tests {
        tc := tc // pre 1.22: capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // safe since tc is captured
            got := Add(tc.a, tc.b)
            if got != tc.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.expected)
            }
        })
    }
}
```

---

## 4. testify Patterns

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
    // require — stops test on failure (use for setup/preconditions)
    svc, err := NewService(cfg)
    require.NoError(t, err)           // test stops if service failed to create
    require.NotNil(t, svc)

    // assert — continues test on failure (use for assertions)
    result, err := svc.Process("input")
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
    assert.Contains(t, result, "partial")

    // Error type checking
    _, err = svc.Process("")
    var ve *ValidationError
    assert.ErrorAs(t, err, &ve)
    assert.Equal(t, "input", ve.Field)
}
```

**Rule:** use `require` for anything that would make subsequent assertions meaningless.
Use `assert` for independent checks.

---

## 5. Mocks with mockgen

```go
// In the interface file — add go:generate directive:
//go:generate mockgen -destination=mocks/user_repo_mock.go -package=mocks . UserRepository

type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

```bash
# Generate all mocks
go generate ./...

# Or directly:
mockgen -destination=mocks/user_repo_mock.go -package=mocks \
  github.com/myorg/myproject/internal/user UserRepository
```

```go
// Using the mock in tests:
import "github.com/myorg/myproject/internal/user/mocks"

func TestService_Process(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    repo := mocks.NewMockUserRepository(ctrl)
    repo.EXPECT().
        FindByID(gomock.Any(), "user-123").
        Return(&User{ID: "user-123", Name: "Alice"}, nil)

    svc := NewService(repo)
    result, err := svc.Process(context.Background(), "user-123")
    require.NoError(t, err)
    assert.Equal(t, "Alice", result.Name)
}
```

---

## 6. Test Helpers

```go
// t.Helper() — marks function as helper so failures point to caller, not helper
func createTestUser(t *testing.T, db *DB, name string) *User {
    t.Helper() // <- this line
    user := &User{Name: name}
    require.NoError(t, db.Save(user))
    return user
}

// t.Cleanup() — runs after test (even on failure), replaces defer
func TestWithDB(t *testing.T) {
    db := connectTestDB(t)
    t.Cleanup(func() { db.Close() })
    // no need for defer db.Close()
}

// t.TempDir() — creates temp dir, auto-cleaned after test
func TestFileProcessor(t *testing.T) {
    dir := t.TempDir() // cleaned automatically
    // write test files to dir
}
```

---

## 7. Coverage Commands

```bash
# Запустить тесты с покрытием
go test -coverprofile=coverage.out ./...

# Покрытие по каждой функции
go tool cover -func=coverage.out

# Итоговый % покрытия
go tool cover -func=coverage.out | tail -1

# HTML-отчёт (открывается в браузере)
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# Проверить конкретный пакет
go test -cover ./internal/user/...

# Покрытие только определённых пакетов (исключить generated)
go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
```

**Thresholds (рекомендуемые):**
- Новый код: ≥ 80%
- Существующий: ≥ 70% (предупреждение), ≥ 50% (нельзя снижать)
- Критические пакеты (auth, payments): ≥ 90%

---

## 8. Race-Safe Tests

```bash
# ВСЕГДА запускать с -race в CI
go test -race ./...

# С -count=1 чтобы избежать кэширования результатов
go test -race -count=1 ./...

# Параллельные тесты — объявить явно:
func TestConcurrent(t *testing.T) {
    t.Parallel() // тест запускается параллельно — должен быть race-safe
    ...
}
```

---

## 9. Build Tags for Integration Tests

```go
//go:build integration
// +build integration

package mypackage_test

// Запуск: go test -tags=integration ./...
// В CI/CD запускать отдельно от unit-tests
```

---

## 10. Key Commands

```bash
# Все тесты с race detector
go test -race -count=1 ./...

# Конкретный тест по имени
go test -run TestGetUser ./internal/user/...

# Конкретный subtest
go test -run "TestGetUser/not_found" ./internal/user/...

# Verbose output
go test -v -race ./...

# Покрытие + race
go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

# Красивый вывод через gotestsum (если установлен)
gotestsum --format=testdox ./...
```
