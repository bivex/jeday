#!/bin/bash
# metrics-report.sh
# Полный отчет по метрикам Go-проекта (аналог und metrics)
# Запуск: bash metrics-report.sh
# Требования: gocyclo, gocognit, gocloc (необязательный)

echo "=============================="
echo "  GO METRICS REPORT"
echo "  $(date '+%Y-%m-%d %H:%M')"
echo "  Dir: $(pwd)"
echo "=============================="

# --- Tool checks ---
for tool in gocyclo gocognit gocloc; do
  if ! which "$tool" > /dev/null 2>&1; then
    echo "WARNING: $tool not installed"
  fi
done
echo ""

echo "--- PACKAGES ---"
PKG_COUNT=$(go list ./... 2>/dev/null | wc -l | tr -d ' ')
echo "Total packages: $PKG_COUNT"
echo ""

echo "--- LINE COUNT (gocloc) ---"
if which gocloc > /dev/null 2>&1; then
  gocloc ./
else
  echo "(gocloc not installed: go install github.com/hhatto/gocloc/cmd/gocloc@latest)"
  # fallback: use find
  TOTAL=$(find . -name "*.go" ! -name "*_test.go" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
  TEST=$(find . -name "*_test.go" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
  echo "Approx production code lines: $TOTAL"
  echo "Approx test code lines: $TEST"
fi
echo ""

echo "--- CYCLOMATIC COMPLEXITY (gocyclo) ---"
if which gocyclo > /dev/null 2>&1; then
  echo "Average:"
  gocyclo -avg ./... 2>/dev/null | tail -1

  echo ""
  echo "Top 10 most complex functions:"
  gocyclo -top 10 ./... 2>/dev/null

  echo ""
  OVER_15=$(gocyclo -over 15 ./... 2>/dev/null | wc -l | tr -d ' ')
  echo "Functions with complexity > 15: $OVER_15"
  if [ "$OVER_15" -gt 0 ]; then
    echo "--- Details ---"
    gocyclo -over 15 ./... 2>/dev/null
  fi
else
  echo "(gocyclo not installed: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)"
fi
echo ""

echo "--- COGNITIVE COMPLEXITY (gocognit) ---"
if which gocognit > /dev/null 2>&1; then
  echo "Top 10 hardest to read:"
  gocognit -top 10 ./... 2>/dev/null

  echo ""
  OVER_25=$(gocognit -over 25 ./... 2>/dev/null | wc -l | tr -d ' ')
  echo "Functions with cognitive complexity > 25: $OVER_25"
  if [ "$OVER_25" -gt 0 ]; then
    echo "--- Refactoring candidates ---"
    gocognit -over 25 ./... 2>/dev/null
  fi
else
  echo "(gocognit not installed: go install github.com/uudashr/gocognit/cmd/gocognit@latest)"
fi
echo ""

echo "=============================="
echo "  THRESHOLDS REFERENCE"
echo "  Cyclomatic: >15 attention, >20 must fix"
echo "  Cognitive:  >15 attention, >25 must fix"
echo "  File LOC:   >500 attention, >1000 must fix"
echo "=============================="
