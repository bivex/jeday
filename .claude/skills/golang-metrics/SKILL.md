---
name: golang-metrics
description: >
  Go code metrics: cyclomatic complexity (gocyclo), cognitive complexity (gocognit),
  lines of code statistics (gocloc), and package-level inventory via go list.
  Use when asked about code complexity, maintainability scores, code size statistics,
  or to find the most complex/largest functions. Equivalent to "und metrics" from SciTools Understand.
allowed-tools: Bash(go *), Bash(gocyclo *), Bash(gocognit *), Bash(gocloc *), Read
---

# Go Code Metrics

This skill covers complexity measurement and code statistics — the Go equivalent of `und metrics`
from SciTools Understand. Go has no single `metrics` command; instead, specialized CLI tools
each measure one dimension.

## Quick Reference

| Und Command | Go Equivalent | Metric |
|-------------|--------------|--------|
| `und metrics` (complexity) | `gocyclo -over N ./...` | Cyclomatic complexity |
| `und metrics` (cognitive) | `gocognit -top N ./...` | Cognitive complexity |
| `und metrics` (LOC) | `gocloc ./` | Lines of code |
| `und metrics` (packages) | `go list -json ./...` | Package inventory |

## Tool Installation

```bash
# Цикломатическая сложность (классика)
which gocyclo  || echo "MISSING: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"

# Когнитивная сложность (более современная)
which gocognit || echo "MISSING: go install github.com/uudashr/gocognit/cmd/gocognit@latest"

# Статистика строк кода
which gocloc   || echo "MISSING: go install github.com/hhatto/gocloc/cmd/gocloc@latest"
```

## Workflow A — Cyclomatic Complexity (gocyclo)

Cyclomatic complexity counts the number of linearly independent paths through code.
**Rule of thumb:** > 10 = needs attention, > 20 = must refactor.

```bash
# Все функции с цикломатической сложностью > 10
gocyclo -over 10 ./...

# Показать топ-15 самых сложных функций
gocyclo -top 15 ./...

# Только конкретная директория
gocyclo -over 10 ./cmd/...
gocyclo -over 10 ./internal/...

# Включая тестовые файлы
gocyclo -over 10 -avg ./...

# Средняя сложность по всем функциям
gocyclo -avg ./...
```

**Interpreting output:**
```
15 mypackage FunctionName /path/to/file.go:42:1
^^ complexity  ^^ function   ^^ location
```

## Workflow B — Cognitive Complexity (gocognit)

Cognitive complexity measures how difficult the code is to read and understand — often a
better proxy for maintainability than cyclomatic complexity.
**Rule of thumb:** > 15 = hard to read, > 25 = must refactor.

```bash
# Топ-10 самых когнитивно сложных функций
gocognit -top 10 ./...

# Функции с когнитивной сложностью > 15
gocognit -over 15 ./...

# Анализ конкретных пакетов
gocognit -top 20 ./internal/...

# Скрипт: найти функции выше порога и показать их расположение
gocognit -over 25 ./... | awk '{print $4": "$2" (complexity "$1")"}' | sort -t: -k1,1
```

**Why both metrics?**
- `gocyclo` catches structural branching but misses nesting depth.
- `gocognit` penalizes deep nesting and breaks in linear flow (early returns, `goto`, recursion).
- For refactoring priorities, use **gocognit** score > cyclomatic score = deeply nested logic.

## Workflow C — Lines of Code Statistics (gocloc)

```bash
# Общая статистика по всему проекту
gocloc ./

# Разбивка по файлам (аналог und metrics по файлам)
gocloc --by-file ./

# JSON вывод для парсинга
gocloc --output=json ./ > code-stats.json

# Только .go файлы, исключая тесты
gocloc --include-lang=Go --not-match="_test\.go$" ./

# Топ-20 самых больших файлов
gocloc --by-file --output=json ./ | \
  jq '.files | to_entries | sort_by(-.value.code) | .[0:20] | .[] | {file: .key, lines: .value.code}'
```

**Output columns:**
- `files` — number of files
- `blank` — blank lines
- `comment` — comment lines
- `code` — actual code lines (most important metric)

## Workflow D — Package-Level Inventory (go list)

```bash
# Список всех пакетов (аналог und list)
go list ./...

# Количество пакетов в проекте
go list ./... | wc -l

# JSON-инвентаризация: пакет + зависимости + файлы
go list -json ./... | jq '{
  package: .ImportPath,
  files: (.GoFiles | length),
  deps: (.Imports | length),
  testFiles: (.TestGoFiles | length)
}'

# Найти самые "тяжелые" пакеты по числу зависимостей
go list -json ./... | jq -s 'sort_by(-.Imports | length) | .[0:10] | .[] | {pkg: .ImportPath, deps: (.Imports | length)}'
```

## Workflow E — Combined Metrics Report

Run a full metrics sweep and produce a structured summary:

```bash
#!/bin/bash
# go-metrics-report.sh — аналог "und metrics" для всего проекта

echo "=============================="
echo "  GO CODE METRICS REPORT"
echo "  $(date '+%Y-%m-%d %H:%M')"
echo "=============================="
echo ""

echo "--- LINE COUNT (gocloc) ---"
gocloc ./ 2>/dev/null || echo "gocloc not installed"
echo ""

echo "--- TOP 10 CYCLOMATIC COMPLEXITY (gocyclo) ---"
gocyclo -top 10 ./... 2>/dev/null || echo "gocyclo not installed"
echo ""

echo "--- TOP 10 COGNITIVE COMPLEXITY (gocognit) ---"
gocognit -top 10 ./... 2>/dev/null || echo "gocognit not installed"
echo ""

echo "--- FUNCTIONS NEEDING REFACTOR (complexity > 20) ---"
echo "Cyclomatic > 20:"
gocyclo -over 20 ./... 2>/dev/null || echo "(none or gocyclo not installed)"
echo ""
echo "Cognitive > 25:"
gocognit -over 25 ./... 2>/dev/null || echo "(none or gocognit not installed)"
echo ""

echo "--- PACKAGE COUNT ---"
go list ./... | wc -l | xargs -I{} echo "{} packages total"
```

## Thresholds Reference Table

| Metric | Good | Acceptable | Needs Attention | Must Refactor |
|--------|------|------------|-----------------|---------------|
| Cyclomatic (gocyclo) | ≤ 5 | 6–10 | 11–20 | > 20 |
| Cognitive (gocognit) | ≤ 8 | 9–15 | 16–25 | > 25 |
| File LOC | ≤ 200 | 200–500 | 500–1000 | > 1000 |
| Package deps | ≤ 5 | 6–10 | 11–15 | > 15 |

## Supporting Files

- See [examples/metrics-report.sh](examples/metrics-report.sh) for the full report script.
