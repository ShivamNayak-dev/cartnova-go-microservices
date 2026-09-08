# CartNova — Event-Driven E-Commerce Microservices Platform

CartNova is a practical, production-oriented e-commerce backend built with Go. It is designed as a reverse-engineering project: every major technology has a concrete architectural purpose, and the code is intentionally written in normal, readable multi-line Go rather than compressed or generated-looking one-liners.

## Why this project exists

CartNova is designed to demonstrate the backend skills expected from a Go developer working with microservices:

- Go REST APIs
- gRPC service-to-service communication
- Apache Kafka event-driven workflows
- Redis caching
- PostgreSQL transactions
- MongoDB document storage
- WebSockets
- JWT authentication and authorization
- Goroutines, channels and controlled concurrency
- Worker pools, fan-in/fan-out and pipelines
- Context cancellation and timeouts
- Race detection and concurrency testing
- Docker and Docker Compose
- Practical AWS deployment planning

The project deliberately contains only five business services plus an API Gateway. It avoids unnecessary microservices and infrastructure that would make the project harder to finish or explain.

## Architecture

```text
                              Client
                                |
                               REST
                                |
                                v
                         +---------------+
                         |  API Gateway  |
                         +-------+-------+
                                 |
          +----------------------+----------------------+
          |                      |                      |
          v                      v                      v
   +-------------+        +-------------+        +-------------+
   | User Service|        |Product Svc  |        | Order Svc   |
   +------+------+        +------+------+        +------+------+
          |                      |                      |
          v                      v                      v
     PostgreSQL             PostgreSQL                PostgreSQL
                                |
                              Redis
                                                       |
                                                      Kafka
                                                 +-----+-----+
                                                 |           |
                                                 v           v
                                        +------------+ +-------------+
                                        | Inventory  | |Notification |
                                        |  Service   | |   Service   |
                                        +-----+------+ +------+------+
                                              |               |
                                          PostgreSQL        MongoDB
                                                              |
                                                          WebSocket
                                                              |
                                                            Client
```

## Services

### 1. API Gateway

Single public entry point. It performs JWT validation, routing and WebSocket proxying.

### 2. User Service

Handles registration, login, profiles and roles. Uses PostgreSQL. A local ADMIN account is seeded for development.

### 3. Product Service

Handles products and categories. Uses PostgreSQL for source-of-truth data and Redis for product-detail caching.

### 4. Order Service

Owns orders and order items. It uses gRPC to validate users/products and publishes order events to Kafka.

### 5. Inventory Service

Owns stock and reservations. It consumes order events concurrently and uses PostgreSQL row locking and transactions to prevent overselling.

### 6. Notification Service

Consumes order events, stores notification documents in MongoDB and pushes notifications to connected users through WebSockets.

## Order workflow

A typical order follows this path:

```text
POST /api/v1/orders
        |
        v
   API Gateway
        |
        v
   Order Service
        |
        | gRPC
        +----> User Service
        |
        +----> Product Service
        |
        | create order transaction
        v
     PostgreSQL
        |
        | OrderCreated
        v
      Kafka
      /   \
     /     \
    v       v
Inventory  Notification
    |          |
    |          +----> MongoDB
    |
    | InventoryReserved / InventoryReservationFailed
    v
  Kafka
    |
    v
Order Service
    |
    +----> CONFIRMED or CANCELLED
    |
    v
  Kafka
    |
    v
Notification Service
    |
    v
 WebSocket
    |
    v
  Client
```

## Concurrency design

Concurrency is not added merely to put the word "concurrency" in the README.

The Inventory Service is deliberately designed for concurrent event processing. Multiple Kafka consumers can process different partitions, and each consumer uses goroutines and a jobs channel around its processing loop.

Inventory correctness is still owned by PostgreSQL. A Go mutex protects only memory inside one process; it cannot protect inventory when multiple service instances are running.

The inventory transaction locks the product row with `FOR UPDATE`, verifies stock, updates available/reserved quantities and records the reservation atomically.

The repository also includes focused concurrency examples under `examples/concurrency` so individual concepts can be reverse-engineered independently before reading the distributed workflow.

## Concurrency topics covered

- Goroutines
- Unbuffered channels
- Buffered channels
- Channel closing
- `range` over channels
- `select`
- `sync.WaitGroup`
- `sync.Mutex`
- `sync.RWMutex`
- Worker pools
- Producer/consumer
- Fan-in/fan-out
- Pipelines
- `context.Context`
- Cancellation
- Timeouts
- Race conditions
- Deadlocks
- Goroutine leaks
- Data-race detection
- `go test -race ./...`

Not every concept is forced into a business feature. Concepts that would be artificial in production are demonstrated in focused examples and tests.

## Technology stack

| Area | Technology |
|---|---|
| Language | Go 1.23 |
| Public API | REST / `net/http` |
| Internal RPC | gRPC |
| Messaging | Apache Kafka |
| Cache | Redis |
| Relational storage | PostgreSQL 17 |
| Document storage | MongoDB 8 |
| Real-time | WebSockets |
| Authentication | JWT + bcrypt |
| Containerization | Docker + Docker Compose |
| Cloud | AWS deployment documentation |
| Testing | Go testing + race detector |

## Project structure

```text
cartnova/
├── cmd/
│   ├── gateway/
│   ├── user-service/
│   ├── product-service/
│   ├── order-service/
│   ├── inventory-service/
│   └── notification-service/
│
├── internal/
│   ├── auth/
│   ├── concurrency/
│   ├── config/
│   ├── database/
│   ├── events/
│   ├── grpcapi/
│   ├── httpx/
│   ├── logging/
│   ├── user/
│   ├── product/
│   ├── order/
│   ├── inventory/
│   └── notification/
│
├── migrations/
├── proto/
├── examples/concurrency/
├── docs/
├── deployments/
├── scripts/
├── web/
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Local service ports

| Component | HTTP | gRPC |
|---|---:|---:|
| API Gateway | 8080 | - |
| User Service | 8081 | 9091 |
| Product Service | 8082 | 9092 |
| Order Service | 8083 | - |
| Inventory Service | 8084 | - |
| Notification Service | 8085 | - |
| Kafka | 29092 | - |
| PostgreSQL | 5432 | - |
| Redis | 6379 | - |
| MongoDB | 27017 | - |

## Local prerequisites

- Go 1.23+
- Docker Desktop
- Docker Compose
- Git
- `curl`
- Python 3 for the smoke-test JSON extraction

## Run with Docker

1. Copy `.env.example` to `.env`.
2. Set a real `JWT_SECRET`.
3. Start everything:

```bash
docker compose up --build -d
```

4. Check the gateway:

```bash
curl http://localhost:8080/health
```

5. Run the smoke test:

```bash
make smoke
```

To reset all databases and Kafka data during development:

```bash
docker compose down -v
docker compose up --build -d
```

The `-v` command deletes local development data, so do not use it against important environments.

## Run Go code directly

Start infrastructure only:

```bash
docker compose up -d postgres redis mongodb kafka kafka-init
```

Then download dependencies and run the services individually:

```bash
go mod tidy

go run ./cmd/user-service
go run ./cmd/product-service
go run ./cmd/order-service
go run ./cmd/inventory-service
go run ./cmd/notification-service
go run ./cmd/gateway
```

Run each command in a separate terminal. The default local ports are documented in `.env.example` and the service source.

## Testing

```bash
go test ./...
```

Run the race detector:

```bash
go test -race ./...
```

Format everything:

```bash
make fmt
```

## Development credentials

PostgreSQL:

```text
user: postgres
password: root
```

MongoDB:

```text
user: root
password: root
```

These are local development credentials only.

Seeded CartNova admin:

```text
email: admin@cartnova.local
password: password
```

Change/remove this before any real deployment.

## API documentation

See:

- `docs/api.md`
- `docs/events.md`
- `docs/concurrency.md`
- `docs/architecture.md`

## Reverse-engineering study order

When studying the project, do not read everything at once. Use this order:

1. Read the architecture document.
2. Start with one service, preferably User Service.
3. Follow `handler -> service -> repository -> database`.
4. Trace a Product request and then Redis cache behavior.
5. Trace Order Service and its gRPC clients.
6. Trace `OrderCreated` into Kafka.
7. Trace Inventory's concurrent processing.
8. Inspect the PostgreSQL transaction and `FOR UPDATE` lock.
9. Trace Inventory's result event back to Order Service.
10. Trace Notification Service and MongoDB.
11. Trace WebSocket delivery.
12. Study the concurrency examples separately.
13. Run the race detector.
14. Break one component intentionally and observe the failure path.
15. Read Docker Compose last so the infrastructure makes sense after the application flow is understood.

## Project design principle

Every major technology has a reason:

- REST: public API contract
- gRPC: low-overhead synchronous internal calls
- Kafka: asynchronous domain events
- Redis: product read caching
- PostgreSQL: transactional business data
- MongoDB: notification documents
- WebSockets: real-time user updates
- Goroutines/channels: controlled concurrent processing
- Docker: reproducible local infrastructure
- AWS: deployment target

The project intentionally does not include Kubernetes, service mesh, distributed tracing platforms or other infrastructure unless a later real requirement justifies them.
