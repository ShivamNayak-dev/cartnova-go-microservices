# Failure Scenarios to Study

CartNova is intentionally suitable for failure-oriented reverse engineering.

## Kafka unavailable

Order creation can fail while trying to publish `OrderCreated`. This exposes an important distributed-systems limitation: a database transaction and a Kafka publish are separate operations.

A future production enhancement would be the transactional outbox pattern.

## Redis unavailable

Product Service should still be able to read from PostgreSQL. Cache failures are treated as cache misses.

## Inventory database unavailable

The inventory consumer cannot complete the reservation and should not acknowledge the Kafka message until the business operation is successfully handled.

## Duplicate event

Consumers use event IDs to avoid intentionally applying the same business event repeatedly.

## Client disconnects

The WebSocket connection is removed from the notification hub.

## Request cancellation

`context.Context` propagates cancellation to downstream work and background processing.

## Concurrent stock requests

PostgreSQL row locking prevents two transactions from both consuming the same available stock.
