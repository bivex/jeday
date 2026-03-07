# Jeday Auth Service 🚀

High-performance, secure authentication service built with Go, following the **Jedi Architecture** principles.

## 🌟 Key Features

- **Blazing Fast Registration**: Best recent local registration-only benchmark reached **~41k RPS**; the current tuned defaults deliver stable mid-30k RPS on full reset-and-profile runs.
- **Asynchronous Password Hardening**: Background workers upgrade weak hashes to **Argon2id** (OWASP recommended) without affecting user-facing latency.
- **High-Performance Web Framework**: Powered by **Atreugo** (fasthttp-based) with **Prefork** mode enabled.
- **Hot Path Optimized for Writes**: Registration uses real bulk insert into `users` plus queue enqueue into `password_upgrade_queue`.
- **Worker Isolation**: The hardening worker runs with a smaller DB pool and tighter CPU/memory limits to reduce contention with the API.
- **Secure Token Management**: Uses **Paseto** (Platform-Agnostic Security Tokens) instead of JWT for improved security.
- **Clean Architecture**: Strict separation of concerns (Domain, Application, Infrastructure, Delivery).
- **Database Integrity**: Schema-first approach with **sqlc** and **golang-migrate**.
- **Observability**: Structured JSON logging with **zerolog** and built-in **pprof** for profiling.

## 🛠 Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: [Atreugo](https://github.com/savsgio/atreugo) (High performance)
- **Database**: PostgreSQL 16
- **Token**: Paseto V2
- **Hashing**: Argon2id + SHA-256 (Fast Path)
- **Migrations**: golang-migrate
- **SQL Gen**: sqlc
- **Load Testing**: k6

## 🏗 Architecture: The "Jedi Maneuver"

1.  **Request**: User sends registration data.
2.  **Fast Path**: API computes a fast SHA-256 hash (`v1$...`) and batches registration requests.
3.  **Bulk Write**: The batcher does a real bulk insert into `users` and enqueues weak hashes into `password_upgrade_queue`.
4.  **Response**: The API returns the created user immediately; password hardening stays off the request path.
5.  **Hardening**: A background `worker` polls `password_upgrade_queue`, re-hashes with `Argon2id` (64 MiB, 1 iteration, 4 threads), writes the strong hash into `user_passwords`, and removes the queue row.
6.  **Smart Login**: Login looks up the user by `email` and verifies either the strong hash or the pending weak hash, depending on upgrade state.

## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose

### Run the Stack
```sh
docker compose up --build -d
```

### Health Check
```sh
curl -sf http://localhost:8080/health
```

### Run Load Tests
```sh
# k6 does NOT auto-start on `docker compose up`; it is kept out of the default stack

# Registration stress test used in recent profiling runs
docker compose run --rm -v $(pwd)/load-test-reg-only.js:/load-test.js k6 run --vus 500 --duration 20s /load-test.js

# One-shot reset + rebuild + eBPF/k6 profile run with artifacts
./scripts/profile_k6.sh full
```

## ⚙️ Current Tuned Defaults

- `SERVER_PREFORK=true`
- `APP_DB_POOL_MAX_CONNS=128`
- `POSTGRES_MAX_CONNECTIONS=256`
- `POSTGRES_SHARED_BUFFERS=512MB`
- `POSTGRES_CHECKPOINT_TIMEOUT=15min`
- `POSTGRES_CHECKPOINT_COMPLETION_TARGET=0.95`
- `POSTGRES_WAL_BUFFERS=64MB`
- `POSTGRES_EFFECTIVE_IO_CONCURRENCY=32`
- `PGBOUNCER_DEFAULT_POOL_SIZE=100`
- `REGISTRATION_BATCH_SIZE=100`
- `REGISTRATION_BATCH_WAIT=10ms`
- worker isolation remains enabled (`WORKER_DB_POOL_MAX_CONNS=4`, `WORKER_GOMAXPROCS=1`, `WORKER_GOMEMLIMIT=256MiB`)

## 📈 Performance (Local Benchmarks)

Recent measured results on local Docker Compose runs:

| Scenario | Registration RPS | Avg Latency | p95 Latency |
| :--- | :--- | :--- | :--- |
| Before the hot-path fixes (worker contending with API) | ~3,875.8/s | n/a | n/a |
| Clean-slate optimized stack | **34,023.7/s** | **14.56ms** | **21.89ms** |
| After dropping DB-level `username` uniqueness | **34,557.6/s** | **14.32ms** | **20.67ms** |
| Best measured sweep peak after `CopyFrom(users)` plus tuned API DB pool (`pool_max_conns=128`) | **41,227.2/s** | **11.87ms** | **22.41ms** |
| After reducing Postgres `max_connections` to `256` (with API pool `128`) | **35,156.0/s** | **14.05ms** | **20.53ms** |
| Current tuned defaults incl. heavy Postgres combo, confirmatory `full` run | **35,170.0/s** | **14.03ms** | **20.89ms** |
| Same setup under eBPF profiling | **31,975.0/s** | **15.38ms** | **24.75ms** |

Notes:
- Numbers above are from registration-only `k6` runs (`500 VUs`, `20s`) against the local Docker Compose stack.
- Current tuned default for the API path is `APP_DB_POOL_MAX_CONNS=128`; larger defaults like `500` reduced throughput noticeably in the tuning sweep.
- A later confirmatory A/B still kept `128` ahead of `500` (`35,885.0/s` vs `33,456.5/s`), even though absolute numbers varied between runs.
- Current tuned Postgres default is `POSTGRES_MAX_CONNECTIONS=256`; in isolated A/B it beat the previous `1000` setting (`35,156.0/s` vs `34,339.9/s`).
- Additional `shared_buffers` / `checkpoint_timeout` experiments did **not** beat the current baseline, so defaults remain `512MB` and `15min`.
- Heavy Postgres frontier tuning was fixed to `checkpoint_completion_target=0.95`, `wal_buffers=64MB`, and `effective_io_concurrency=32`; confirmatory A/B showed a repeatable gain over the prior baseline, though local Docker Desktop runs still show visible variance.
- `email` remains unique and is used for login lookup.
- `username` is **currently not unique at the DB level**; removing `users_username_key` reduced write cost on the registration path.

## 📁 Project Structure

```text
.
├── cmd/
│   ├── api/          # Entry point for the Atreugo API
│   └── worker/       # Entry point for the Hardening Worker
├── internal/
│   ├── auth/         # Auth domain logic
│   │   ├── delivery/ # HTTP handlers & middleware
│   │   ├── repository/ # Database interactions
│   │   ├── service/  # Business logic (Usecases)
│   │   └── token/    # Paseto & Hashing implementation
│   ├── config/       # Configuration management
│   └── db/           # Generated sqlc code
├── migrations/       # SQL migration files
├── query.sql         # SQL queries for sqlc
└── sqlc.yaml         # sqlc configuration
```

## 🛡 Security Notes
- **Paseto** eliminates common JWT pitfalls (algorithm confusion, etc.).
- **Argon2id** is memory-hard and side-channel resistant.
- **Prefork** isolation: each worker process handles its own memory space.
- **Current trade-off**: login identity is `email`; `username` is currently treated as display data, not as a unique login identifier.
