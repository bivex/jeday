---
name: golang-concurrency
description: >
  Go concurrency patterns, context propagation, goroutine lifecycle, channel ownership,
  sync primitives, goroutine leak prevention, and race detector usage.
  Use when writing or reviewing concurrent Go code.
allowed-tools: Bash(go test -race *), Bash(go vet *), Bash(golangci-lint *), Read, Grep
---

# Go Concurrency Safety

Concurrency is the most dangerous area in Go. Data races cause undefined behavior.
Goroutine leaks exhaust memory. This skill enforces safe patterns from the start.

## The Three Laws of Go Concurrency

1. **Every goroutine must have a defined exit condition** — no fire-and-forget without lifecycle control
2. **context.Context is the cancellation bus** — propagate it, never store it, always check it
3. **Channels have a single owner** — only the sender closes, never the receiver

---

## 1. context.Context Rules

### ✅ Correct patterns

```go
// 1. Context is ALWAYS the first parameter
func ProcessOrder(ctx context.Context, orderID string) error { ... }

// 2. With timeout at the entry point (HTTP handler, main, job)
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel() // ALWAYS defer cancel — prevents context leak
    
    result, err := processOrder(ctx, r.URL.Query().Get("id"))
    ...
}

// 3. Propagate to all downstream calls
func processOrder(ctx context.Context, id string) error {
    if err := db.QueryContext(ctx, ...); err != nil { return err }
    if err := cache.SetContext(ctx, ...); err != nil { return err }
    return nil
}

// 4. Check for cancellation in long loops
func processItems(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        if err := process(ctx, item); err != nil { return err }
    }
    return nil
}
```

### ❌ Anti-patterns

```go
// WRONG: storing context in struct
type Service struct {
    ctx context.Context // NEVER do this
}

// WRONG: context.Background() inside a function that received ctx
func (s *Service) DoWork(ctx context.Context) {
    go doThing(context.Background()) // drops cancellation chain!
}

// WRONG: missing cancel — context leak
ctx, _ := context.WithTimeout(parent, 5*time.Second) // MISSING defer cancel()

// WRONG: using context.TODO() in production code (only for WIP)
func ProductionHandler(ctx context.Context) {
    doSomething(context.TODO()) // should pass ctx
}
```

---

## 2. Goroutine Lifecycle

### ✅ sync.WaitGroup — basic fan-out

```go
func processAll(ctx context.Context, items []Item) error {
    var wg sync.WaitGroup
    errs := make(chan error, len(items)) // buffered — no goroutine leak

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := process(ctx, item); err != nil {
                errs <- err
            }
        }(item) // capture loop variable — important pre Go 1.22
    }

    wg.Wait()
    close(errs)

    for err := range errs {
        if err != nil { return err }
    }
    return nil
}
```

### ✅ errgroup — structured concurrency with errors (preferred)

```go
import "golang.org/x/sync/errgroup"

func processAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx) // ctx cancelled if any goroutine fails

    for _, item := range items {
        item := item // pre 1.22: capture loop variable
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait() // blocks until all done, returns first non-nil error
}
```

### ✅ Worker pool (bounded concurrency)

```go
func workerPool(ctx context.Context, jobs <-chan Job, workers int) error {
    g, ctx := errgroup.WithContext(ctx)

    for i := 0; i < workers; i++ {
        g.Go(func() error {
            for {
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case job, ok := <-jobs:
                    if !ok { return nil } // channel closed
                    if err := process(ctx, job); err != nil { return err }
                }
            }
        })
    }

    return g.Wait()
}
```

### ❌ Goroutine anti-patterns

```go
// WRONG: fire-and-forget — no way to know when it finishes or if it errored
go doWork()

// WRONG: goroutine without exit condition — will run forever
go func() {
    for {
        process()
        // no ctx.Done() check, no channel to stop
    }
}()

// WRONG: passing loop variable by reference (pre Go 1.22)
for _, item := range items {
    go func() {
        process(item) // BUG: all goroutines see last value of item
    }()
}
```

---

## 3. Channel Ownership & Patterns

### ✅ Ownership rules

```go
// Producer owns the channel: creates it, writes to it, closes it
func producer(ctx context.Context) <-chan int { // returns receive-only
    ch := make(chan int, 10) // buffered reduces blocking
    go func() {
        defer close(ch) // producer closes
        for i := 0; i < 100; i++ {
            select {
            case <-ctx.Done(): return
            case ch <- i:
            }
        }
    }()
    return ch
}

// Consumer only reads
func consumer(ctx context.Context, ch <-chan int) {
    for {
        select {
        case <-ctx.Done(): return
        case v, ok := <-ch:
            if !ok { return } // channel closed
            use(v)
        }
    }
}
```

### ❌ Channel anti-patterns

```go
// WRONG: closing from receiver
func badConsumer(ch chan int) {
    close(ch) // panic if producer still writing
}

// WRONG: unbuffered channel in hot path — sender blocks until receiver ready
results := make(chan Result) // causes tight coupling
```

---

## 4. sync Primitives

### sync.Mutex / sync.RWMutex

```go
type SafeCache struct {
    mu    sync.RWMutex
    store map[string]string
}

func (c *SafeCache) Get(key string) (string, bool) {
    c.mu.RLock()         // read lock — concurrent reads OK
    defer c.mu.RUnlock()
    v, ok := c.store[key]
    return v, ok
}

func (c *SafeCache) Set(key, value string) {
    c.mu.Lock()          // write lock — exclusive
    defer c.mu.Unlock()
    c.store[key] = value
}
```

### sync.Pool — reduce allocations in hot paths

```go
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func processRequest(data []byte) string {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()
    buf.Write(data)
    return buf.String()
}
```

### sync.Once — safe initialization

```go
type Singleton struct {
    once     sync.Once
    instance *DB
}

func (s *Singleton) DB() *DB {
    s.once.Do(func() {
        s.instance = connectDB()
    })
    return s.instance
}
```

---

## 5. Race Detection Commands

```bash
# Запуск тестов с детектором гонок (ОБЯЗАТЕЛЬНО в CI)
go test -race ./...

# Сборка с детектором (для ручного тестирования)
go build -race -o app_race .

# Статический анализ на concurrency issues
go vet ./...
golangci-lint run --enable=govet,staticcheck --disable-all ./...

# Поиск go func() без контроля жизненного цикла
grep -rn "go func(" --include="*.go" . | grep -v "_test.go"

# Поиск WaitGroup без Add (возможный баг)
grep -rn "\.Wait()" --include="*.go" . | grep -v "_test.go"
```

---

## 6. Goroutine Leak Detection

```bash
# Установить goleak для тестов
go get go.uber.org/goleak

# В тестах:
# func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
# func TestMyFunc(t *testing.T) { defer goleak.VerifyNone(t) ... }

# Проверить количество горутин в запущенном сервисе
curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -50
```

---

## Quick Anti-Pattern Checklist

| Check | Command |
|-------|---------|
| Data races | `go test -race ./...` |
| Goroutine fire-and-forget | `grep -rn "^go " --include="*.go" .` |
| Missing cancel() | `grep -rn "WithTimeout\|WithCancel\|WithDeadline" --include="*.go" . ` |
| Context in struct | `grep -rn "ctx.*context.Context" --include="*.go" . \| grep "struct {"` |
| Channel closed by receiver | Code review only |
