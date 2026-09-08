# CartNova Concurrency Design

Concurrency is a first-class learning and implementation goal in CartNova.

## Goroutines

Long-running consumers and service loops run in goroutines so the HTTP server and background processing can operate independently.

## Channels

The Inventory Service uses channels as the boundary between Kafka fetching and worker processing. The standalone `internal/concurrency.WorkerPool` also uses a jobs channel.

## Worker pool

Inventory processing is controlled by `WORKER_COUNT` rather than creating an unlimited goroutine per Kafka event.

## Database consistency

A Go mutex cannot protect data across multiple service instances. Inventory correctness therefore uses PostgreSQL transactions and `SELECT ... FOR UPDATE` row locking.

The database transaction:

1. checks an existing reservation,
2. locks the inventory row,
3. verifies available stock,
4. decrements available stock,
5. increments reserved stock,
6. records the reservation,
7. commits atomically.

## Idempotency

Consumers record processed event IDs. Duplicate Kafka deliveries do not intentionally create duplicate business effects.

## Context

HTTP requests and background consumers use `context.Context` for cancellation and shutdown propagation.

## Race detection

Run:

```bash
go test -race ./...
```

## Learning examples

`examples/concurrency` contains focused examples for:

- goroutines
- channels
- buffered channels
- select
- WaitGroup
- Mutex
- worker pools
- fan-in/fan-out
- pipelines
- context cancellation
