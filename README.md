# API Rate Limiter

A gRPC-based rate limiting service written in Go. The project supports multiple rate limiting algorithms, optional Redis-backed state for shared/distributed enforcement, and optional PostgreSQL-backed per-key configuration overrides.

## Features

- gRPC API for request admission decisions
- Two algorithms:
  - `token_bucket`
  - `rolling_window`
- Two limiter state backends:
  - `inmemory`
  - `redis`
- Optional PostgreSQL config store for per-key limits
- Docker and `docker-compose` setup
- Basic smoke client and unit tests

## How It Works

Clients call the `Check` RPC with:

- `key`: identifier to rate limit on, such as a user ID, API key, or IP
- `limit`: maximum requests/tokens
- `window_seconds`: window size used by the algorithm

The server returns:

- `decision`: `ALLOW` or `DENY`
- `remaining`: remaining requests/tokens
- `retry_after_ms`: when to retry after a denial

If PostgreSQL configuration is enabled and a matching row exists for the request `key`, the stored `limit` and `window_seconds` override the values sent by the client.

## Project Structure

```text
.
├── cmd/smoke-client/               # Simple gRPC client for manual testing
├── gen/ratelimiter/v1/             # Generated protobuf and gRPC code
├── migrations/                     # PostgreSQL schema
├── proto/ratelimiter/v1/           # Protobuf contract
├── src/
│   ├── token_bucket.go             # In-memory token bucket limiter
│   ├── rolling_window.go           # In-memory rolling window limiter
│   ├── redis_limiter.go            # Redis-backed limiter using Lua scripts
│   ├── config_store.go             # PostgreSQL config store
│   └── *_test.go                   # Unit/integration tests
├── Dockerfile
├── docker-compose.yml
└── server.go                       # gRPC server entrypoint
```

## Requirements

- Go `1.25.5`
- Docker and Docker Compose for containerized runs
- Redis if using `LIMITER_BACKEND=redis`
- PostgreSQL if using `DATABASE_URL`

## Running Locally

### 1. Install dependencies

```bash
go mod download
```

### 2. Start the server

Default mode uses the in-memory token bucket limiter:

```bash
go run ./server.go
```

The server listens on `:50051`.

### 3. Run the smoke client

In another terminal:

```bash
go run ./cmd/smoke-client
```

Expected output looks like:

```text
decision=ALLOW remaining=2 retry_after_ms=0
```

## Configuration

The service is configured entirely through environment variables.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `LIMITER_BACKEND` | No | `inmemory` | Limiter state backend: `inmemory` or `redis` |
| `LIMITER_ALGORITHM` | No | `token_bucket` | Algorithm: `token_bucket` or `rolling_window` |
| `REDIS_ADDR` | Only for Redis | none | Redis address, for example `localhost:6379` |
| `DATABASE_URL` | No | none | PostgreSQL connection string for loading per-key config |

### Example: in-memory rolling window

```bash
LIMITER_BACKEND=inmemory \
LIMITER_ALGORITHM=rolling_window \
go run ./server.go
```

### Example: Redis token bucket

```bash
LIMITER_BACKEND=redis \
LIMITER_ALGORITHM=token_bucket \
REDIS_ADDR=localhost:6379 \
go run ./server.go
```

### Example: Redis + PostgreSQL config store

```bash
LIMITER_BACKEND=redis \
LIMITER_ALGORITHM=rolling_window \
REDIS_ADDR=localhost:6379 \
DATABASE_URL='postgres://ratelimiter:ratelimiter@localhost:5432/ratelimiter?sslmode=disable' \
go run ./server.go
```

## PostgreSQL Configuration Overrides

When `DATABASE_URL` is set, the server loads rate limit configuration from the `rate_limit_configs` table.

Schema:

```sql
CREATE TABLE IF NOT EXISTS rate_limit_configs (
    key TEXT PRIMARY KEY,
    limit_value INTEGER NOT NULL CHECK (limit_value > 0),
    window_seconds INTEGER NOT NULL CHECK (window_seconds > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Example row:

```sql
INSERT INTO rate_limit_configs (key, limit_value, window_seconds)
VALUES ('demo-client', 5, 60)
ON CONFLICT (key) DO UPDATE
SET limit_value = EXCLUDED.limit_value,
    window_seconds = EXCLUDED.window_seconds,
    updated_at = NOW();
```

If a client sends a request for `demo-client`, the server will use `5 requests / 60 seconds` regardless of the values sent in the RPC payload.

## Running with Docker Compose

The repository includes a ready-to-run stack with:

- gRPC server
- Redis
- PostgreSQL

Start everything:

```bash
docker compose up --build
```

This exposes:

- gRPC server on `localhost:50051`
- Redis on `localhost:6379`
- PostgreSQL on `localhost:5432`

The compose setup configures:

- `LIMITER_BACKEND=redis`
- `DATABASE_URL=postgres://ratelimiter:ratelimiter@postgres:5432/ratelimiter?sslmode=disable`

If you want to switch algorithms in Compose, update `LIMITER_ALGORITHM` in `docker-compose.yml`.

## gRPC API

Protobuf definition: `proto/ratelimiter/v1/ratelimiter.proto`

### Request

```proto
message RateLimitRequest {
  string key = 1;
  uint32 limit = 2;
  uint32 window_seconds = 3;
}
```

### Response

```proto
message RateLimitResponse {
  enum Decision {
    DECISION_UNSPECIFIED = 0;
    ALLOW = 1;
    DENY = 2;
  }

  Decision decision = 1;
  uint32 remaining = 2;
  uint64 retry_after_ms = 3;
}
```

### RPC

```proto
service RateLimiter {
  rpc Check(RateLimitRequest) returns (RateLimitResponse);
}
```

## Algorithms

### Token Bucket

- Each key has a bucket with a maximum token capacity equal to `limit`
- Tokens refill continuously over `window_seconds`
- A request is allowed when at least one token is available
- `retry_after_ms` indicates when the next token should be available

### Rolling Window

- Each key stores timestamps for recent accepted requests
- Requests older than `window_seconds` are discarded
- A request is allowed if the number of requests still inside the window is below `limit`
- `retry_after_ms` indicates when the oldest accepted request will expire from the window

## Testing

Run unit tests:

```bash
go test ./...
```

Notes:

- Redis-backed tests automatically skip if Redis is not reachable at `127.0.0.1:6379`
- In-memory tests cover both token bucket and rolling window behavior

## Example Development Flow

1. Start the server with the backend/algorithm you want to validate.
2. Run `go run ./cmd/smoke-client`.
3. Repeat the client call to observe `ALLOW`, `remaining`, and eventual `DENY` behavior.
4. If using PostgreSQL overrides, insert a row for the same `key` and call again.

## License

No license file is currently included in the repository.
