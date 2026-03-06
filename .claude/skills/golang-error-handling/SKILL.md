---
name: golang-error-handling
description: >
  Go error handling patterns: %w wrapping, errors.Is/As, sentinel errors, custom error types,
  panic prohibition rules, and error API design. Use when writing or reviewing error handling code.
allowed-tools: Bash(go vet *), Bash(golangci-lint *), Bash(grep *), Read, Grep
---

# Go Error Handling Patterns

Go's explicit error handling is its greatest strength for reliability — when done correctly.
This skill enforces idiomatic, chain-preserving, API-stable error handling.

---

## 1. The Golden Rules

| Rule | Why |
|------|-----|
| Always use `%w` to wrap | Preserves chain for `errors.Is`/`errors.As` |
| Never use `panic` in libraries/services | Caller cannot recover gracefully |
| Use `errors.Is` not `==` for comparison | Works through wrap chain |
| Use `errors.As` not type assertion | Works through wrap chain |
| Export sentinel errors only when needed | Avoid tight coupling |

---

## 2. Error Wrapping

### ✅ Correct: `%w` preserves the chain

```go
// Wrap with context — %w is key, not %v
func loadUser(id string) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, fmt.Errorf("loadUser %s: %w", id, err) // %w wraps
    }
    return user, nil
}

// Caller can inspect the original error
if err := loadUser("42"); err != nil {
    if errors.Is(err, sql.ErrNoRows) { // works even through layers of %w
        return ErrUserNotFound
    }
    return fmt.Errorf("handleRequest: %w", err)
}
```

### ❌ Wrong: `%v` breaks the chain

```go
// %v converts to string — errors.Is cannot inspect it
return nil, fmt.Errorf("loadUser: %v", err) // BREAKS chain — DON'T do this
```

### Convention for wrapping messages

```
"operation name: %w"     ✅
"operation name failed: %w"  ✅ (common but redundant)
"failed to do X: %w"     ✅

// Chain reads bottom-up:
// "handleHTTP: loadUser 42: db.Find: connection refused"
```

---

## 3. Sentinel Errors

Sentinel errors signal a specific, expected condition that callers should handle.

```go
// ✅ Correct: package-level sentinel
var ErrNotFound = errors.New("not found")
var ErrPermissionDenied = errors.New("permission denied")

// Caller pattern:
if errors.Is(err, ErrNotFound) {
    http.Error(w, "user not found", http.StatusNotFound)
    return
}
```

### When to expose sentinels

| Expose | Don't expose |
|--------|-------------|
| Caller needs to branch on it | Only logged and returned |
| Different HTTP status per error | Internal implementation detail |
| Part of documented public API | Transient/infrastructure errors |

```go
// ✅ Unexported sentinel — internal use only
var errRetryExceeded = errors.New("max retries exceeded")

// ✅ Exported — callers need to handle it
var ErrInvalidInput = errors.New("invalid input")
```

---

## 4. Custom Error Types

Use custom types when callers need structured data from the error.

```go
// ✅ Typed error with fields
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: field %q: %s", e.Field, e.Message)
}

// Raise:
return nil, &ValidationError{Field: "email", Message: "must contain @"}

// Inspect with errors.As (works through wrap chain):
var ve *ValidationError
if errors.As(err, &ve) {
    http.Error(w, ve.Message, http.StatusUnprocessableEntity)
}

// ✅ Wrapping preserves As detection:
return nil, fmt.Errorf("createUser: %w", &ValidationError{...})
```

---

## 5. errors.Is and errors.As

```go
// errors.Is — check if error IS a specific value (through chain)
if errors.Is(err, io.EOF) { ... }
if errors.Is(err, ErrNotFound) { ... }

// errors.As — extract typed error (through chain)
var netErr *net.OpError
if errors.As(err, &netErr) {
    fmt.Println(netErr.Op, netErr.Addr)
}

// NEVER use == for non-sentinel comparison
if err == someErr { ... }         // ❌ breaks if wrapped
if errors.Is(err, someErr) { ... } // ✅
```

---

## 6. panic Rules

```go
// ✅ ONLY acceptable uses of panic:
// 1. Programmer invariant violations (impossible state):
func mustPositive(n int) {
    if n <= 0 { panic(fmt.Sprintf("mustPositive: got %d", n)) }
}

// 2. Package init failures (program cannot start):
func init() {
    tmpl = template.Must(template.ParseFiles("tmpl.html")) // panics on bad template
}

// 3. Test helpers (panic becomes test failure):
func mustMarshal(t *testing.T, v any) []byte {
    t.Helper()
    b, err := json.Marshal(v)
    if err != nil { t.Fatalf("marshal: %v", err) }
    return b
}

// ❌ NEVER in library/service code:
func GetUser(id string) *User {
    u, err := db.Find(id)
    if err != nil { panic(err) } // WRONG — caller cannot handle
    return u
}
```

---

## 7. Detection Commands

```bash
# Find unhandled errors
golangci-lint run --enable=errcheck --disable-all ./...

# Find broken wrap chain (%v instead of %w)
golangci-lint run --enable=errorlint --disable-all ./...

# Find missing error wrapping (wrapcheck — optional, strict)
golangci-lint run --enable=wrapcheck --disable-all ./...

# Find bare panic calls in production code
grep -rn "panic(" --include="*.go" . | grep -v "_test.go" | grep -v "//.*panic"

# Find == comparison on errors (should use errors.Is)
grep -rn "== err\b\|err ==" --include="*.go" . | grep -v "_test.go"

# Find %v used to wrap errors (should be %w)
grep -rn 'Errorf.*%v.*err' --include="*.go" .
```

---

## 8. golangci-lint config for error handling

```yaml
# In .golangci.yml
linters:
  enable:
    - errcheck      # unhandled errors
    - errorlint     # incorrect wrapping patterns  
    - wrapcheck     # errors from external packages must be wrapped

linters-settings:
  errcheck:
    check-type-assertions: true  # x.(T) without ok
    check-blank: true            # _ = f()
  wrapcheck:
    ignorePackageGlobs:
      - "github.com/myorg/myproject/*"  # internal errors don't need wrapping
```
