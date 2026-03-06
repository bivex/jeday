---
name: golang-project-structure
description: >
  Go Standard Project Layout: /cmd, /internal, /pkg conventions, import organization,
  goimports, circular dependency prevention, and module boundaries.
  Use when scaffolding or reviewing Go project structure.
allowed-tools: Bash(go list *), Bash(go mod *), Bash(goda *), Bash(goimports *), Bash(gofmt *), Read, Grep, Glob
---

# Go Standard Project Layout

A well-structured Go project makes the architecture self-documenting and prevents
a class of coupling bugs before they happen.

---

## 1. Standard Layout

```
myproject/
├── cmd/                    # Entry points (main packages only)
│   ├── server/
│   │   └── main.go         # go build ./cmd/server
│   └── worker/
│       └── main.go         # go build ./cmd/worker
│
├── internal/               # Private code — cannot be imported by external modules
│   ├── user/               # Domain: user
│   │   ├── service.go
│   │   ├── service_test.go
│   │   ├── repository.go   # interface definition
│   │   └── mocks/
│   ├── order/
│   └── platform/           # Cross-cutting: DB, HTTP, cache
│       ├── database/
│       ├── httpserver/
│       └── cache/
│
├── pkg/                    # Public reusable packages (use sparingly)
│   └── validator/          # Only if external projects will import this
│
├── api/                    # API contracts: .proto, OpenAPI, JSON schema
│   └── openapi.yaml
│
├── configs/                # Config templates (not secrets)
│   └── config.yaml.example
│
├── scripts/                # Build/analysis/migration scripts
│
├── go.mod
├── go.sum
└── README.md
```

---

## 2. Rules per Directory

### `/cmd` — only `main` packages

```go
// ✅ Correct: thin main.go — just wires dependencies
package main

func main() {
    cfg := config.Load()
    db  := platform.NewDB(cfg.Database)
    svc := user.NewService(db)
    srv := httpserver.New(cfg.HTTP, svc)
    srv.Run()
}

// ❌ Wrong: business logic in main.go
func main() {
    rows, err := db.Query("SELECT * FROM users") // business logic in main
    ...
}
```

### `/internal` — the most important directory

```
- Enforced by the Go compiler: packages in /internal can only be imported
  by code rooted at the parent of /internal.
- Put ALL business logic, domain models, and implementation details here.
- Prefer internal/ over pkg/ by default.
```

```go
// This import is FORBIDDEN from external modules:
import "github.com/myorg/myproject/internal/user" // compiler error outside myproject
```

### `/pkg` — only truly public, stable APIs

Use `/pkg` ONLY when:
- Other repositories will import this package
- The API is stable and versioned
- It has no knowledge of the business domain

---

## 3. Import Organization (goimports)

Three groups, separated by blank lines:

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "net/http"

    // 2. External dependencies
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    // 3. Internal packages
    "github.com/myorg/myproject/internal/user"
    "github.com/myorg/myproject/pkg/validator"
)
```

```bash
# Check formatting
goimports -l ./...

# Apply formatting (modifies files)
goimports -w ./...

# Install goimports
go install golang.org/x/tools/cmd/goimports@latest
```

---

## 4. Circular Dependency Prevention

Go forbids circular imports at compile time. Architecture must be layered:

```
cmd → internal/domain → internal/platform → stdlib
```

**Dependency direction rules:**
- `cmd` imports `internal/*` — allowed
- `internal/domain` imports `internal/platform` — allowed
- `internal/platform` imports `internal/domain` — FORBIDDEN (cycle)
- Two domain packages import each other — FORBIDDEN (cycle)

### Breaking cycles: Interface Injection

```go
// ❌ Cycle: user imports order, order imports user

// ✅ Solution: define interface in the consuming package
// internal/user/service.go
type OrderQuerier interface {         // defined here, implemented in /order
    GetRecentOrders(ctx context.Context, userID string) ([]Order, error)
}

type Service struct {
    orders OrderQuerier // inject the abstraction
}
```

### Detection

```bash
# Import cycle = compile error
go build ./... 2>&1 | grep -i "import cycle"

# Dependency tree to spot direction violations
goda tree ./...:all

# List all imports of a package
go list -json ./internal/user/... | jq '.Imports[]'

# Check which packages import a specific package
go list -json ./... | jq 'select(.Imports[]? == "github.com/myorg/myproject/internal/user") | .ImportPath'
```

---

## 5. Package Naming Rules

```
✅ user, order, payment, cache, database
✅ httpserver, grpcserver (descriptive compound)
✅ validator, formatter, parser

❌ util, utils, common, helpers, misc, shared
❌ userutil, orderhelper (redundant suffix)
❌ UserService (exported name as package — use user.Service)
❌ user_service (underscore — only in test files)
```

**Rule:** name a package by what it provides, not what it contains.

---

## 6. Verification Commands

```bash
# 1. List all packages
go list ./...

# 2. Dependency tree (spot direction violations)
goda tree ./...:all

# 3. Cycle detection
go build ./... 2>&1 | grep -i cycle

# 4. Import formatting
goimports -l ./...

# 5. Check internal boundary (packages outside internal/ should not import internal/)
# (enforced by compiler — if it builds, it's OK)

# 6. Find packages named util/common/helpers (anti-pattern)
go list ./... | grep -E "util|common|helper|misc|shared"
```
