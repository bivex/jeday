---
name: golang-profiling
description: >
  Go performance profiling workflows: CPU profiling, memory profiling, benchmarking,
  goroutine/block/mutex profiling, execution tracing, and pprof analysis.
  Use when asked to profile performance, analyze memory usage, find CPU hotspots,
  run/compare benchmarks, or investigate slow Go code.
allowed-tools: Bash(go test *), Bash(go tool pprof *), Bash(go tool trace *), Bash(go build *), Bash(go run *), Bash(benchstat *), Bash(curl *), Read, Grep
---

# Go Performance Profiling

The complete CLI toolkit for Go performance analysis — no external platforms required.

## Quick Reference

| Task | Command |
|------|---------|
| Run benchmarks | `go test -bench=. -benchmem ./...` |
| CPU profile | `go test -bench=. -cpuprofile=cpu.out ./...` |
| Memory profile | `go test -bench=. -memprofile=mem.out ./...` |
| Analyze profile (text) | `go tool pprof -top cpu.out` |
| Analyze profile (web UI) | `go tool pprof -http=:8080 cpu.out` |
| Compare benchmarks | `benchstat old.txt new.txt` |
| Block profile | `go test -bench=. -blockprofile=block.out ./...` |
| Mutex contention | `go test -bench=. -mutexprofile=mutex.out ./...` |
| Execution trace | `go test -trace=trace.out ./...` |
| Analyze trace | `go tool trace trace.out` |

---

## Workflow 1 — Benchmarks (Baseline)

Always start with benchmarks to get reproducible numbers before profiling.

```bash
# Запуск всех бенчмарков с метриками выделения памяти
go test -bench=. -benchmem ./...

# Фильтр по имени функции (regexp)
go test -bench=BenchmarkMyFunc -benchmem ./pkg/mypackage/...

# N повторений для стабильных результатов (минимум 5)
go test -bench=. -benchmem -count=5 ./...

# Сохранить результаты для сравнения
go test -bench=. -benchmem -count=5 ./... | tee bench-before.txt
```

**Метрики в выводе:**
```
BenchmarkSort-8    500000    2453 ns/op    128 B/op    3 allocs/op
                   ^runs     ^time/op      ^memory     ^allocations
```

### Сравнение до/после (benchstat)

```bash
# Установка benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Сохранить результаты ДО изменений
go test -bench=. -benchmem -count=10 ./... > bench-before.txt

# [внести изменения в код]

# Сохранить результаты ПОСЛЕ
go test -bench=. -benchmem -count=10 ./... > bench-after.txt

# Статистическое сравнение
benchstat bench-before.txt bench-after.txt
```

**Пример вывода benchstat:**
```
name        old time/op    new time/op    delta
Sort-8        2.45µs ± 2%    1.87µs ± 1%  -23.67%  (p=0.000 n=10+10)

name        old allocs/op  new allocs/op  delta
Sort-8          3.00 ± 0%      1.00 ± 0%  -66.67%  (p=0.000 n=10+10)
```

---

## Workflow 2 — CPU Profiling

Найти функции, потребляющие больше всего CPU.

### Сбор профиля

```bash
# Через тесты/бенчмарки (самый простой способ)
go test -bench=BenchmarkMyFunc -cpuprofile=cpu.out ./pkg/mypackage/

# Через запуск программы (добавить в main: defer pprof.StopCPUProfile())
go build -o myapp . && ./myapp --cpuprofile=cpu.out

# Через HTTP endpoint (для работающего сервиса с net/http/pprof)
curl -s "http://localhost:6060/debug/pprof/profile?seconds=30" -o cpu.out
```

### Анализ CPU профиля

```bash
# Топ функций по CPU (текстовый вывод)
go tool pprof -top cpu.out

# Топ-20 функций
go tool pprof -top20 cpu.out

# С учётом только своего кода (исключить стандартную библиотеку)
go tool pprof -top -nodecount=20 cpu.out

# Граф вызовов в DOT (текст)
go tool pprof -dot cpu.out | head -50

# Интерактивный веб-интерфейс (открыть в браузере http://localhost:8080)
go tool pprof -http=:8080 cpu.out

# Источник: показать аннотированный исходный код
go tool pprof -source MyFunction cpu.out
```

**Интерпретация `-top` вывода:**
```
Showing nodes accounting for 4.2s, 91% of 4.6s total
      flat  flat%   sum%        cum   cum%
     1.82s 39.57% 39.57%      1.82s 39.57%  runtime.mallocgc
     0.93s 20.22% 59.78%      0.93s 20.22%  runtime.memmove

# flat = время внутри функции (без вызываемых ею функций)
# cum  = суммарное время (включая вызовы)
# Большой cum при малом flat → функция вызывает медленную подфункцию
```

---

## Workflow 3 — Memory Profiling

Найти утечки памяти и избыточные аллокации.

### Сбор профиля

```bash
# Через тесты
go test -bench=BenchmarkMyFunc -memprofile=mem.out ./pkg/mypackage/

# Через HTTP endpoint
curl -s "http://localhost:6060/debug/pprof/heap" -o mem.out

# Аллокации (не heap — чистый счётчик запросов памяти)
curl -s "http://localhost:6060/debug/pprof/allocs" -o allocs.out
```

### Анализ памяти

```bash
# Топ функций по heap-использованию (по умолчанию inuse_space)
go tool pprof -top mem.out

# Топ по количеству аллокаций
go tool pprof -alloc_objects -top mem.out

# Топ по объёму аллоцированной памяти (не только живой)
go tool pprof -alloc_space -top mem.out

# Сравнение двух heap-профилей (разница = утечка)
go tool pprof -top -base mem-before.out mem-after.out

# Интерактивный UI
go tool pprof -http=:8081 mem.out
```

---

## Workflow 4 — Goroutine & Blocking Profiling

Найти блокировки, дедлоки, goroutine leaks.

### Goroutine profiling

```bash
# Снимок текущих горутин (для работающего сервиса)
curl -s "http://localhost:6060/debug/pprof/goroutine?debug=2" | head -100

# Сохранить и проанализировать
curl -s "http://localhost:6060/debug/pprof/goroutine" -o goroutine.out
go tool pprof -top goroutine.out
```

### Block profiling (блокировки channel/mutex)

```bash
# Включить в коде перед запуском: runtime.SetBlockProfileRate(1)
go test -bench=. -blockprofile=block.out ./...
go tool pprof -top block.out
```

### Mutex contention

```bash
# Включить в коде: runtime.SetMutexProfileFraction(1)
go test -bench=. -mutexprofile=mutex.out ./...
go tool pprof -top mutex.out
```

---

## Workflow 5 — Execution Trace

Детальная трассировка планировщика Go, GC, syscalls.

```bash
# Сбор трейса (до 10 секунд, обычно достаточно)
go test -trace=trace.out ./pkg/mypackage/ -run TestMyFunc

# Для работающего сервиса (5 секунд)
curl -s "http://localhost:6060/debug/pprof/trace?seconds=5" -o trace.out

# Открыть в браузере
go tool trace trace.out
```

В UI трейса смотреть на:
- **Goroutine Analysis** → долгое ожидание (Sync Block)  
- **GC pauses** → частые STW = много мелких аллокаций  
- **Syscall** → много времени в ядре = IO bottleneck

---

## Workflow 6 — Live pprof (работающий сервис)

Если в коде есть `import _ "net/http/pprof"` и HTTP-сервер на `:6060`:

```bash
# Все доступные профили
curl -s http://localhost:6060/debug/pprof/

# Быстрый текстовый дамп heap
curl -s "http://localhost:6060/debug/pprof/heap?debug=1" | head -40

# 30-секундный CPU профиль живого сервиса
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Горутины с трассировками стека
curl -s "http://localhost:6060/debug/pprof/goroutine?debug=2"
```

Добавить в `main.go` для включения:
```go
import _ "net/http/pprof"

// В main():
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

---

## Интерпретация результатов

### Красные флаги CPU

| Симптом | Вероятная причина |
|---------|------------------|
| `runtime.mallocgc` в топе | Слишком много мелких аллокаций |
| `runtime.gcBgMarkWorker` | GC занят → много живых объектов |
| `sync.(*Mutex).Lock` | Высокая конкуренция за мьютекс |
| `syscall.Read/Write` | IO bottleneck |
| `reflect.*` в топе | Избыточное использование рефлексии |

### Красные флаги памяти

| Симптом | Вероятная причина |
|---------|------------------|
| Растущий heap между снимками | Утечка памяти |
| Много аллокаций в hot path | Используйте `sync.Pool` или prealloc |
| Большие объекты в heap | Уйти от копирования, передавать указатели |
| `strings.Builder` не используется | Конкатенация строк в цикле |

---

## Supporting Files

- See [examples/profile-report.sh](examples/profile-report.sh) for a complete profiling script.
