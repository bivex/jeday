---
name: golang-security
description: >
  Go security scanning: govulncheck, gosec, dependency vulnerabilities,
  SQL injection prevention, hardcoded secrets, and unsafe usage.
  Use when answering security questions or scanning Go projects.
allowed-tools: Bash(govulncheck *), Bash(gosec *), Bash(go mod *), Bash(golangci-lint *), Bash(grep *), Read, Grep
---

# Go Security Scanning & Hardening

Security in Go requires proactive dependency auditing (`govulncheck`), static analysis (`gosec`),
and strict discipline regarding external input, secrets, and the `unsafe` package.

---

## 1. Vulnerability DB (govulncheck)

The official Go vulnerability database tracks CVEs in the standard library and third-party modules.

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan the current project (source code)
govulncheck ./...

# Scan a compiled binary
govulncheck -mode=binary /path/to/binary

# Output JSON for CI/CD integration
govulncheck -json ./...
```

**Rule:** Zero CRITICAL or HIGH vulnerabilities allowed in production.
Use `go get u package@version` to patch specific vulnerable dependencies.

---

## 2. Static Application Security Testing (gosec)

`gosec` scans Go ASTs for common security pitfalls (SQLi, hardcoded credentials).

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Scan the whole project
gosec ./...

# Excluding tests (recommended)
gosec -exclude-dir=test -exclude-generated ./...

# Output text format (easier to read in CLI)
gosec -fmt=text ./...
```

### Key gosec Rules:
- **G101**: Look for hardcoded credentials (passwords, secrets)
- **G104**: Audit errors not checked
- **G201**, **G202**: SQL query construction using format strings (SQL injection)
- **G304**: File path provided as taint input (directory traversal)
- **G401**: Detect the usage of DES, RC4, MD5 or SHA1 (weak crypto)
- **G402**: TLS InsecureSkipVerify set true
- **G501**: Import blocklist: crypto/md5
- **G601**: Implicit memory aliasing in for loop (Go < 1.22)

---

## 3. Secret Hygiene

Secrets (passwords, API keys, tokens) must never be committed to source code or logged.

### ✅ Correct Pattern (Environment/Config)

```go
func ConnectDB() *sql.DB {
    password := os.Getenv("DB_PASSWORD")
    if password == "" { log.Fatal("DB_PASSWORD is required") }
    // pass securely
}
```

### ❌ Anti-Pattern (Hardcoded)

```go
// NEVER DO THIS
const dbPass = "super_secret_password_123"

// NEVER LOG SECRETS
log.Printf("Connecting with token %s", apiKey) // WRONG
```

### Detection Commands

```bash
# Regex search for common secret variable names (except in tests)
grep -inE "password|secret|token|api_key" $(find . -name "*.go" | grep -v "_test.go")

# Using golangci-lint
golangci-lint run --enable=gosec ./...
```

---

## 4. Input Validation & SQL Safety

All external input (HTTP requests, env vars, files) is untrusted until validated.

### ✅ Parameterized Queries (Safe)

```go
// The database driver handles escaping
rows, err := db.Query("SELECT * FROM users WHERE name = $1", req.Name)
```

### ❌ String Interpolation (SQL Injection)

```go
// NEVER construct SQL dynamically with user input
query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", req.Name) // G201
rows, err := db.Query(query)
```

### Validation Libraries

Encourage the use of validation libraries for complex struct validation:
- `github.com/go-playground/validator/v10`
- `github.com/ozontech/allure-go/pkg/framework/asserts_wrapper`

---

## 5. Unsafe Code (`unsafe` and `reflect`)

The `unsafe` package bypasses Go's type safety and memory management.

### Rules for `unsafe`
1. Never use `unsafe` unless absolutely required for CGO or extreme performance optimization (proven by benchmarks).
2. If used, it must be isolated in a dedicated internal package.
3. Every use of `uintptr` must be carefully audited.

### Detection

```bash
# Find imports of the unsafe package
grep -rn '"unsafe"' --include="*.go" .
```

---

## 6. Module Integrity

Ensure dependencies haven't been tampered with.

```bash
# Verify module checksums against go.sum
go mod verify

# Download/update module dependencies
go mod download
```

If `go mod verify` fails, the downloaded module contents differ from the expected checksum. This could indicate a compromised dependency or a proxy cache issue.

---

## 7. Security Scanning Workflow

Run these commands in order during a security review:

1. `govulncheck ./...` (Checks for known CVEs)
2. `gosec -fmt=text ./...` (Checks for vulnerable code patterns)
3. `go mod verify` (Checks dependency integrity)
4. Manual review of `unsafe` imports and package boundaries.
