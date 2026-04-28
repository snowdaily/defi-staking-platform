# Backend

Go services for the staking protocol.

## Services

| Binary | Purpose |
|--------|---------|
| `cmd/indexer` | Subscribes to chain events, mirrors state to Postgres |
| `cmd/api` | REST + GraphQL API serving frontend |
| `cmd/rewardbot` | Cron-driven reward distribution to vault |

## Setup

Requires Go 1.21+ and PostgreSQL 15+.

```bash
go mod tidy
docker compose up -d postgres
make migrate
```

## Run

```bash
go run ./cmd/indexer
go run ./cmd/api
go run ./cmd/rewardbot
```

## Test

```bash
go test ./...                    # Unit tests
go test -tags=integration ./...  # Integration tests (uses testcontainers)
```
