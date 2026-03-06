---
name: golang-architecture
description: >
  Go package dependency analysis and call graph visualization using goda, go-callvis,
  go mod graph, and go list. Use when asked to visualize architecture, understand package
  relationships, find circular dependencies, or map call flows in a Go project.
  Equivalent to "und export -dependencies" from SciTools Understand.
allowed-tools: Bash(go *), Bash(goda *), Bash(go-callvis *), Bash(dot *), Read
---

# Go Architecture & Dependency Analysis

This skill handles dependency graphs, call graphs, and package structure analysis — the Go equivalent
of `und export -dependencies` and the Architecture Diagrams view in SciTools Understand.

## Quick Reference

| Und Command | Go Equivalent |
|-------------|--------------|
| `und export -dependencies` | `goda list ./...:all` (текст) / `goda tree ./...:all` (дерево) |
| Architecture Diagrams | `go-callvis` (SVG/PNG) / `goda graph \| dot` (SVG) |
| Dependency Weight | `goda cut ./...:all` |
| Dependency Tree | `go mod graph` / `goda tree ./...:all` |
| `und list files` | `go list ./...` |
| `und list functions` | `go list -json ./...` |

## Tool Installation Check

```bash
# goda — package dependency analyzer
which goda || echo "MISSING: go install github.com/loov/goda@latest"

# go-callvis — call graph visualizer
which go-callvis || echo "MISSING: go install github.com/ofabry/go-callvis@latest"

# dot — graphviz renderer (for PNG/SVG from DOT files)
which dot || echo "MISSING: brew install graphviz  (macOS) / apt install graphviz (Linux)"
```

## Workflow A — Package Dependency Analysis (goda)

> **Важно:** `goda graph` выводит текст в формате DOT (для программ типа graphviz).
> Для **читаемого вывода в терминале** используйте `goda list` и `goda tree`.

### A1. Список зависимостей (текст)

```bash
# Плоский список всех зависимостей проекта — самый простой вариант
goda list ./...:all

# Только прямые зависимости конкретного пакета
goda list github.com/myorg/myproject/cmd/server:...

# Отфильтровать только внешние (не стандартные) зависимости
goda list ./...:all | grep -v "^std"
```

### A2. Дерево зависимостей (текст, наглядная структура)

```bash
# Дерево зависимостей всего проекта в терминале
goda tree ./...:all

# Дерево конкретного пакета
goda tree github.com/myorg/myproject/cmd/server:...
```

### A3. Анализ веса зависимостей

```bash
# Какие пакеты занимают больше всего места в бинарнике
goda cut ./...:all

# Вес конкретного бинарника (если уже собран)
goda weight ./path/to/binary
```

### A4. Поиск циклических зависимостей

```bash
# Через go list — найти пакеты с ошибками (включая import cycles)
go list -json -e ./... | jq -r 'select(.Error != null) | "\(.ImportPath): \(.Error.Err)"'

# Быстрая проверка через go build (import cycles = ошибка компиляции)
go build ./... 2>&1 | grep -i "import cycle"
```

### A5. Экспорт в SVG/PNG (только если нужна картинка)

```bash
# goda graph выводит DOT-формат → передать в dot для рендеринга
goda graph ./... > deps.dot

# Конвертация в SVG (предпочтительно — масштабируемый)
dot -Tsvg deps.dot -o deps.svg
open deps.svg      # macOS
xdg-open deps.svg  # Linux

# Или PNG
dot -Tpng deps.dot -o deps.png
```

## Workflow B — Call Graph (go-callvis)

### B1. Basic Call Graph

```bash
# Визуализация графа вызовов для main пакета
# Замените 'github.com/myorg/myproject' на ваш модуль из go.mod
MODULE=$(go list -m)
go-callvis -group pkg,dom -file callgraph.svg $MODULE

# Граф для конкретного пакета (не main)
go-callvis -group pkg,dom -focus github.com/myorg/myproject/internal/service -file service-calls.svg $MODULE
```

### B2. Filtered Call Graph

```bash
# Исключить стандартную библиотеку (только проектный код)
go-callvis -nostd -group pkg,dom -file callgraph-noext.svg $MODULE

# Только внутренние пакеты
go-callvis -include "github.com/myorg/myproject" -group pkg -file internal-only.svg $MODULE

# Граф вызовов с группировкой по типу
go-callvis -group type,pkg -file type-graph.svg $MODULE
```

## Workflow C — Module Dependency Tree

```bash
# Встроенный граф зависимостей модулей Go (go.mod уровень)
go mod graph

# С фильтрацией (только прямые зависимости)
go mod graph | grep "^$(go list -m) " | sort

# Граф модулей в DOT (через внешний парсер)
go mod graph | awk '{ print $1 " -> " $2 }' | sort -u | \
  awk 'BEGIN { print "digraph {" } { print "  \"" $1 "\" -> \"" $3 "\"" } END { print "}" }' > modules.dot
dot -Tsvg modules.dot -o modules.svg
```

## Workflow D — Project Inventory (go list)

```bash
# Список всех пакетов в проекте (аналог und list files)
go list ./...

# Подробная информация в JSON (импорты, экспорт, ошибки)
go list -json ./...

# Только пакеты с ошибками компиляции
go list -json -e ./... | jq 'select(.Error != null) | {pkg: .ImportPath, err: .Error.Err}'

# Все внешние зависимости проекта
go list -json ./... | jq -r '.Imports[]' | sort -u | grep -v "^$(go list -m)"

# Статистика по импортам каждого пакета
go list -json ./... | jq '{pkg: .ImportPath, imports: (.Imports | length)}'
```

## Workflow E — Code Documentation (go doc)

```bash
# Документация конкретной функции (аналог und report)
go doc net/http.ListenAndServe

# Полная документация пакета
go doc -all ./internal/service

# Все экспортируемые символы в проекте
go list ./... | xargs -I{} go doc {} 2>/dev/null | grep "^func\|^type\|^var\|^const" | head -100
```

## Reading Architecture Output

When interpreting `goda tree` / `goda list` output:

- Каждая строка в `goda list` — один пакет с его полным import path
- В `goda tree` вложенность показывает транзитивные зависимости
- Пакет, встречающийся глубоко во многих ветках `goda tree` — общая зависимость (potential bottleneck)
- Packages с длинной цепочкой в `goda cut` — кандидаты на замену более лёгкими альтернативами
- Cycles (`A → B → A`) = ошибка `import cycle not allowed` при сборке — критично

Key architectural questions to answer with these tools:
1. **Cycles?** — `go build ./... 2>&1 | grep -i cycle`
2. **Heaviest deps?** — `goda cut ./...:all`
3. **Dependency depth?** — `goda tree ./...:all` (depth of nesting)
4. **Isolation check** — are `internal/` packages only referenced from within their parent?
5. **Layering** — does the flow respect `cmd → internal → pkg` without back-references?

## Supporting Files

- See [examples/architecture-analysis.sh](examples/architecture-analysis.sh) for a complete analysis script.
