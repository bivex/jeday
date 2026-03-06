---
name: golang-compliance
description: >
  Go code quality and compliance analysis using golangci-lint, staticcheck, go vet, and revive.
  Use when asked to check code quality, find linting violations, audit Go code for correctness,
  or when the user asks "what's wrong with this code?" for a Go project.
allowed-tools: Bash(go *), Bash(golangci-lint *), Bash(staticcheck *), Bash(revive *), Read, Grep, Glob
---

# Go Compliance & Quality Analysis

Detailed workflow for running compliance checks on Go projects — the equivalent of `und codecheck` from SciTools Understand.

## Quick Reference

| Und Command | Go Equivalent |
|-------------|--------------|
| `und codecheck "Config"` | `golangci-lint run` |
| `und analyze` | `go build ./...` + `golangci-lint run` |
| `und codecheck -exitstatus` | `golangci-lint run` (exits 1 on issues) |

## Step 1 — Environment Check

Before running any analysis, verify the required tools are installed:

```bash
# Check golangci-lint
which golangci-lint || echo "MISSING: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Check staticcheck
which staticcheck || echo "MISSING: go install honnef.co/go/tools/cmd/staticcheck@latest"

# Check revive
which revive || echo "MISSING: go install github.com/mgechev/revive@latest"
```

## Step 2 — Module Readiness

```bash
# Ensure module metadata is current
go mod verify
go mod tidy
```

## Step 3 — Build Validation (Compile Pass)

```bash
# Aналог первичного анализа: выявить ошибки компиляции
go build -v ./...

# Если нужно проверить конкретный пакет:
go build -v ./cmd/...
```

## Step 4 — Standard Vet (Built-in)

```bash
# go vet — встроенный анализатор Go, всегда запускать первым
go vet ./...

# С расширенными проверками shadow, printf и т.д.:
go vet -shadow ./...
```

## Step 5 — Full Lint Sweep (golangci-lint)

```bash
# Полная проверка с дефолтными линтерами
golangci-lint run ./...

# Вывод ошибок в текстовом формате (по умолчанию)
golangci-lint run --out-format=tab ./...

# JSON-отчет для последующей обработки
golangci-lint run --out-format=json ./... > lint-report.json

# Топ-10 файлов с наибольшим количеством проблем
golangci-lint run --out-format=json ./... | \
  jq '[.Issues[] | .Pos.Filename] | group_by(.) | map({file: .[0], count: length}) | sort_by(-.count)[:10]'
```

## Step 6 — Deep Static Analysis (staticcheck)

```bash
# Глубокие проверки: неиспользуемый код, ошибочные паттерны
staticcheck ./...

# Только определённые категории проверок
staticcheck -checks="SA*" ./...   # ошибки correctness
staticcheck -checks="S*" ./...    # simplifications
staticcheck -checks="ST*" ./...   # style
staticcheck -checks="QF*" ./...   # quickfixes
```

## Step 7 — Style Rules (revive)

```bash
# Современный линтер стиля (настраиваемые правила)
revive -formatter friendly ./...

# Только критические замечания
revive -formatter friendly -set_exit_status ./...
```

## Configuration: .golangci.yml

For repeatable compliance runs, create `.golangci.yml` at the project root.
Suggested minimum configuration for architectural compliance:

```yaml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - govet        # официальный анализатор Go
    - staticcheck  # глубокий анализ корректности
    - errcheck     # непроверенные ошибки
    - gosimple     # упрощения кода
    - ineffassign  # неэффективные присваивания
    - unused       # неиспользуемый код
    - gocyclo      # цикломатическая сложность
    - misspell     # опечатки в комментариях
    - revive       # стиль

linters-settings:
  gocyclo:
    min-complexity: 15
  errcheck:
    check-type-assertions: true
    check-blank: true

issues:
  exclude-rules:
    - path: "_test\\.go"
      linters:
        - errcheck
```

## Interpreting Results

After running compliance tools, categorize findings:

**Critical (must fix before merge):**
- `errcheck` — unhandled errors
- `govet` — suspicious code constructs
- `staticcheck SA*` — correctness issues

**Warnings (should fix):**
- `staticcheck S*` — simplification opportunities
- `gocyclo` violations — overly complex functions
- `ineffassign` — wasted assignments

**Suggestions (consider improving):**
- `revive` style violations
- `misspell` corrections
- `staticcheck QF*` — quick fixes

## Equivalent to `und codecheck -exitstatus` for Scripts

```bash
#!/bin/bash
echo "=== Go Compliance Check ==="

echo "1. Build..."
go build ./... || exit 1

echo "2. Vet..."
go vet ./... || exit 1

echo "3. Lint..."
golangci-lint run --timeout 5m || exit 1

echo "=== All checks passed ==="
```

## Supporting Files

- See [examples/golangci-config.yml](examples/golangci-config.yml) for a production-ready config.
