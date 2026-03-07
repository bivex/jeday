# Jeday Auth Service 🚀

High-performance, secure authentication service built with Go, following the **Jedi Architecture** principles.

## 🌟 Key Features

- **Blazing Fast Registration**: Current local registration-only benchmark reaches **~34k RPS** on a clean Docker Compose stack.
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
```

## 📈 Performance (Local Benchmarks)

Recent measured results on local Docker Compose runs:

| Scenario | Registration RPS | Avg Latency | p95 Latency |
| :--- | :--- | :--- | :--- |
| Before the hot-path fixes (worker contending with API) | ~3,875.8/s | n/a | n/a |
| Clean-slate optimized stack | **34,023.7/s** | **14.56ms** | **21.89ms** |
| After dropping DB-level `username` uniqueness | **34,557.6/s** | **14.32ms** | **20.67ms** |
| Same setup under eBPF profiling | **31,975.0/s** | **15.38ms** | **24.75ms** |

Notes:
- Numbers above are from registration-only `k6` runs (`500 VUs`, `20s`) against the local Docker Compose stack.
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
