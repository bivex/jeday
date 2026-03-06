---
name: golang-style-guide
description: >
  Go style and formatting guidelines: gofmt, goimports, naming conventions,
  comment standards, Effective Go idioms, and golangci-lint configuration.
  Use when reviewing or formatting Go code.
allowed-tools: Bash(gofmt *), Bash(goimports *), Bash(golangci-lint *), Bash(revive *), Read, Grep
---

# Go Style Guide & Idioms

Go codebase consistency is paramount. The language relies heavily on conventions and
standardized tooling rather than strict rules enforced by the compiler.
This skill enforces Effective Go and Go Code Review Comments.

---

## 1. Tooling (Continuous Enforcement)

### `gofmt` & `goimports`

Formatting is not a matter of preference in Go; it is strictly defined by `gofmt`.
`goimports` extends `gofmt` by organizing imports into standard library, external, and internal blocks.

```bash
# Check if files need formatting (returns list of files)
goimports -l ./...

# Format and write to files (fix)
goimports -w ./...

# If goimports is not available, gofmt is acceptable for formatting only
gofmt -s -w ./...
```

### `golangci-lint` (Style-specific linters)

```yaml
# Recommended style-focused configuration (.golangci.yml)
linters:
  enable:
    - revive       # Fast, configurable linter (replacement for golint)
    - godot        # Checks if comments end in a period
    - misspell     # Finds commonly misspelled words
    - unconvert    # Removes unnecessary type conversions
    - goconst      # Finds repeated strings that could be constants
    - gofmt        # Enforces gofmt integration
    - goimports    # Enforces import ordering

linters-settings:
  revive:
    rules:
      - name: exported
      - name: var-naming
      - name: package-comments
      - name: error-naming
      - name: error-strings
      - name: if-return
```

---

## 2. Naming Conventions

### Package Names

- **Short, concise, single word.**
- **Lowercase only.** No `_` or mixedCaps.
- **Descriptive.** Name by what it *provides*, not what it *contains*.
- ❌ `util`, ❌ `common`, ❌ `helpers`, ❌ `misc` (Anti-patterns: meaningless)
- ✅ `http`, ✅ `user`, ✅ `validate`, ✅ `json`

### Variable & Function Names

- **MixedCaps** / **camelCase**. (No `snake_case`).
- **Short scope = short name.** e.g., `i` for index, `r` for Reader, `err` for error.
- **Large scope = descriptive name.** e.g., `userCount`, `connectionString`.
- **Exported vs Unexported:**
  - `Process()` is exposed outside the package.
  - `process()` is private to the package.

### Interface Names

- One-method interfaces typically add an `-er` suffix to the method name.
  - e.g., `Reader`, `Writer`, `Formatter`, `CloseNotifier`.

### Error Variables & Types

- **Variables:** Prefix with `Err`.
  - e.g., `var ErrNotFound = errors.New(...)`
- **Types:** Suffix with `Error`.
  - e.g., `type ValidationError struct { ... }`

---

## 3. Commenting & Documentation (Godoc)

Comments are for documentation. `godoc` expects specific formatting.

### Exported Identifiers Must Be Commented

Every exported (capitalized) name in a program should have a doc comment.

```go
// ✅ Correct:
// Request represents an HTTP request to be sent to a server.
type Request struct { ... }

// DoTheThing performs the main action. It returns an error if...
func DoTheThing() error { ... }

// ❌ Incorrect (redundant, missing subject, or not a full sentence):
// this performs the main action
```

### Package Comments

Packages should have a package comment, introduced with `// Package name...`
It should be immediately before the `package` clause, with no blank line.

```go
// Package math provides basic constants and mathematical functions.
package math
```

---

## 4. Idioms & Best Practices (Effective Go)

### Return Early (Avoid deep nesting)

```go
// ❌ Anti-pattern: deeply nested happy path
func process(user *User) error {
    if user != nil {
        if user.IsActive {
            return doWork(user)
        } else {
            return ErrInactive
        }
    }
    return ErrNilUser
}

// ✅ Idiomatic: guard clauses, return early
func process(user *User) error {
    if user == nil {
        return ErrNilUser
    }
    if !user.IsActive {
        return ErrInactive
    }
    return doWork(user)
}
```

### Zero Values Should Be Usable

Design structs so that their zero value (the default uninitialized state) is safe and useful.
- `sync.Mutex` is ready to use without initialization.
- `bytes.Buffer` is ready to use.

### Accept Interfaces, Return Structs

Functions should accept the smallest interface they require, and return a concrete struct (if applicable).
This provides flexibility for the caller and avoids requiring them to cast interfaces.

```go
// ✅ Accept io.Reader (don't require *os.File), return concrete *User
func LoadUser(r io.Reader) (*User, error) { ... }
```

### Group Related Declarations

Group related variables, constants, or types together.

```go
const (
    StatusActive   = "active"
    StatusInactive = "inactive"
)
```

---

## 5. Verification Commands

```bash
# 1. Format code (Check only)
goimports -l ./...

# 2. Check complete style with golangci-lint
golangci-lint run --enable=revive,godot,misspell,gofmt ./...

# 3. Check for specific anti-patterns (e.g., snake_case variable names)
# (Best caught by revive configuration in golangci-lint)
```
