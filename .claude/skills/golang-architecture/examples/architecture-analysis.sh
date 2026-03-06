#!/bin/bash
# architecture-analysis.sh
# Полный анализ архитектуры Go-проекта (аналог und export -dependencies)
# Запуск: bash architecture-analysis.sh [optional-module-path]
# Требования: goda (обязательно), go-callvis + graphviz (опционально, для SVG)

set -e

MODULE=${1:-$(go list -m 2>/dev/null || echo "")}
if [ -z "$MODULE" ]; then
  echo "ERROR: could not determine Go module name. Run from project root or pass module path."
  exit 1
fi

echo "=============================="
echo "  ARCHITECTURE ANALYSIS"
echo "  Module: $MODULE"
echo "  $(date '+%Y-%m-%d %H:%M')"
echo "=============================="

# --- Tool checks ---
MISSING_GODA=0
if ! which goda > /dev/null 2>&1; then
  echo "WARNING: goda not found"
  echo "  Install: go install github.com/loov/goda@latest"
  MISSING_GODA=1
fi

HAVE_CALLVIS=0
if which go-callvis > /dev/null 2>&1; then HAVE_CALLVIS=1; fi

HAVE_DOT=0
if which dot > /dev/null 2>&1; then HAVE_DOT=1; fi

if [ "$HAVE_CALLVIS" -eq 0 ] || [ "$HAVE_DOT" -eq 0 ]; then
  echo "NOTE: SVG visualizations skipped (optional tools missing)"
  echo "  go install github.com/ofabry/go-callvis@latest"
  echo "  brew install graphviz  # macOS"
fi
echo ""

# --- 1. Package inventory ---
echo "--- 1. PACKAGE LIST ---"
PKG_COUNT=$(go list ./... 2>/dev/null | tee /tmp/_pkglist.txt | wc -l | tr -d ' ')
cat /tmp/_pkglist.txt | sort
echo "(total: $PKG_COUNT packages)"

# --- 2. Flat dependency list (human-readable) ---
echo ""
echo "--- 2. DEPENDENCY LIST (goda list) ---"
if [ "$MISSING_GODA" -eq 0 ]; then
  goda list ./...:all
else
  echo "(skipped — goda not installed)"
fi

# --- 3. Dependency tree (human-readable, best for architecture review) ---
echo ""
echo "--- 3. DEPENDENCY TREE (goda tree) ---"
if [ "$MISSING_GODA" -eq 0 ]; then
  goda tree ./...:all
else
  echo "(skipped — goda not installed)"
fi

# --- 4. Dependency weight (which packages are heaviest) ---
echo ""
echo "--- 4. DEPENDENCY WEIGHT (goda cut) ---"
if [ "$MISSING_GODA" -eq 0 ]; then
  goda cut ./...:all
else
  echo "(skipped — goda not installed)"
fi

# --- 5. Cycle check ---
echo ""
echo "--- 5. CYCLE CHECK ---"
CYCLES=$(go build ./... 2>&1 | grep -i "import cycle" || true)
if [ -z "$CYCLES" ]; then
  echo "✓ No circular dependencies detected (go build clean)"
else
  echo "⚠ IMPORT CYCLES FOUND:"
  echo "$CYCLES"
fi

# Also check via go list for compile-time errors
go list -json -e ./... 2>/dev/null | \
  jq -r 'select(.Error != null) | "  ⚠ \(.ImportPath): \(.Error.Err)"' 2>/dev/null || true

# --- 6. External dependencies ---
echo ""
echo "--- 6. EXTERNAL DEPENDENCIES ---"
go list -json ./... 2>/dev/null | \
  jq -r '.Imports[]' 2>/dev/null | \
  grep '\.' | \
  grep -v "^$(go list -m)" | \
  sort -u | head -40 \
  || go list -json ./... | grep '"[a-z].*\.[a-z]' | sort -u | head -30

# --- 7. SVG visualizations (optional) ---
echo ""
echo "--- 7. VISUALIZATIONS (SVG) ---"
if [ "$MISSING_GODA" -eq 0 ] && [ "$HAVE_DOT" -eq 1 ]; then
  goda graph ./... > /tmp/deps.dot 2>/dev/null
  dot -Tsvg /tmp/deps.dot -o deps.svg 2>/dev/null \
    && echo "✓ deps.svg generated (package dependency graph)" \
    || echo "dot rendering failed"
else
  echo "(skipped — requires goda + graphviz)"
fi

if [ "$HAVE_CALLVIS" -eq 1 ]; then
  go-callvis -nostd -group pkg,dom -file callgraph.svg "$MODULE" 2>/dev/null \
    && echo "✓ callgraph.svg generated (function call graph)" \
    || echo "go-callvis: failed (ensure project builds cleanly)"
else
  echo "(call graph skipped — go-callvis not installed)"
fi

echo ""
echo "=============================="
echo "  ANALYSIS COMPLETE"
echo "=============================="
echo ""
echo "Text output above is readable directly."
echo "SVG files (if generated): deps.svg, callgraph.svg"
