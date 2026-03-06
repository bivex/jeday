#!/bin/bash
# profile-report.sh
# Комплексный профиль производительности Go-проекта
# Запуск: bash profile-report.sh [./path/to/package] [BenchmarkName]
# Требования: стандартный go toolchain; benchstat (опционально)

PKG=${1:-./...}
BENCH=${2:-.}
COUNT=5

echo "=============================="
echo "  GO PERFORMANCE PROFILE"
echo "  Package: $PKG"
echo "  Bench filter: $BENCH"
echo "  $(date '+%Y-%m-%d %H:%M')"
echo "=============================="

# --- Tool checks ---
if ! which benchstat > /dev/null 2>&1; then
  echo "NOTE: benchstat not installed (statistical comparison skipped)"
  echo "  Install: go install golang.org/x/perf/cmd/benchstat@latest"
fi
echo ""

# --- 1. Benchmarks ---
echo "--- 1. BENCHMARKS (go test -bench) ---"
echo "Running $COUNT iterations for statistical stability..."
go test -bench="$BENCH" -benchmem -count="$COUNT" "$PKG" | tee bench-results.txt
echo ""
echo "Results saved to: bench-results.txt"

# --- 2. CPU Profile ---
echo ""
echo "--- 2. CPU PROFILE ---"
# Запускаем только первый совпавший пакет для скорости
FIRST_PKG=$(go list "$PKG" 2>/dev/null | head -1)
if [ -n "$FIRST_PKG" ]; then
  go test -bench="$BENCH" -benchmem -cpuprofile=cpu.out "$FIRST_PKG" > /dev/null 2>&1
  if [ -f cpu.out ]; then
    echo "Top CPU consumers:"
    go tool pprof -top -nodecount=15 cpu.out 2>/dev/null
    echo ""
    echo "Profile saved: cpu.out"
    echo "Interactive UI: go tool pprof -http=:8080 cpu.out"
  else
    echo "(no benchmarks found in $FIRST_PKG — skipping CPU profile)"
  fi
else
  echo "(could not resolve package — skipping CPU profile)"
fi

# --- 3. Memory Profile ---
echo ""
echo "--- 3. MEMORY PROFILE ---"
if [ -n "$FIRST_PKG" ]; then
  go test -bench="$BENCH" -benchmem -memprofile=mem.out "$FIRST_PKG" > /dev/null 2>&1
  if [ -f mem.out ]; then
    echo "Top memory allocators (by allocation count):"
    go tool pprof -alloc_objects -top -nodecount=10 mem.out 2>/dev/null
    echo ""
    echo "Top memory allocators (by size):"
    go tool pprof -alloc_space -top -nodecount=10 mem.out 2>/dev/null
    echo ""
    echo "Profile saved: mem.out"
    echo "Interactive UI: go tool pprof -http=:8081 mem.out"
  else
    echo "(no benchmarks found — skipping memory profile)"
  fi
fi

# --- 4. Block Profile ---
echo ""
echo "--- 4. BLOCK PROFILE (channel/mutex blocking) ---"
if [ -n "$FIRST_PKG" ]; then
  go test -bench="$BENCH" -benchmem -blockprofile=block.out "$FIRST_PKG" > /dev/null 2>&1
  if [ -f block.out ]; then
    echo "Top blocking operations:"
    go tool pprof -top -nodecount=10 block.out 2>/dev/null || echo "(no blocking events recorded)"
  fi
fi

# --- Summary ---
echo ""
echo "=============================="
echo "  PROFILE FILES GENERATED"
echo "=============================="
for f in bench-results.txt cpu.out mem.out block.out; do
  [ -f "$f" ] && echo "  ✓ $f" || true
done
echo ""
echo "NEXT STEPS:"
echo "  1. Open web UI:     go tool pprof -http=:8080 cpu.out"
echo "  2. Compare variants: benchstat bench-before.txt bench-results.txt"
echo "  3. Check goroutines: curl http://localhost:6060/debug/pprof/goroutine?debug=2"
echo "=============================="
