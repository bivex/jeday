# Jeday Auth Service 🚀

High-performance, secure authentication service built with Go, following the **Jedi Architecture** principles.

## 🌟 Key Features

- **Blazing Fast Registration**: Achieved **12,000+ RPS** using a "Fast Path" approach (SHA-256 initial hashing).
- **Asynchronous Password Hardening**: Background workers upgrade weak hashes to **Argon2id** (OWASP recommended) without affecting user-facing latency.
- **High-Performance Web Framework**: Powered by **Atreugo** (fasthttp-based) with **Prefork** mode enabled.
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
2.  **Fast Path**: API computes a fast SHA-256 hash and commits to DB immediately. Latency: **~3-5ms**.
3.  **Response**: User is registered and logged in instantly.
4.  **Hardening**: A background `worker` picks up the weak hash, wraps/re-hashes it using `Argon2id` (64MB memory, 1 iteration, 4 threads), and updates the DB.
5.  **Smart Login**: The system automatically detects if a password is "hardened" or "weak" during login, verifying correctly regardless of the worker's progress.

## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose

### Run the Stack
```sh
docker compose up --build -d
```

### Run Load Tests
```sh
# Health check test
docker compose run k6

# Registration stress test (8k-12k RPS)
docker compose run -v $(pwd)/load-test-reg-only.js:/load-test.js k6 run /load-test.js
```

## 📈 Performance (Local Benchmarks)

| Metric | Original (Argon2 path) | Optimized (Fast Path + Prefork) |
| :--- | :--- | :--- |
| **Registration RPS** | ~100 req/s | **~12,000 req/s** |
| **Latency (p95)** | ~200ms | **~24ms** |
| **Security** | Argon2id | Argon2id (Async) |

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
- **Prefork** isolatation: Each process handles its own memory space.
