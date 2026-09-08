<div align="center">

# 🛒 CartNova

**Event-Driven E-Commerce Microservices Platform in Go**

*A production-style reverse-engineering project covering REST, gRPC, Kafka, Redis, PostgreSQL, MongoDB, WebSockets and real Go concurrency — with a reason behind every piece of infrastructure.*

[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![MongoDB](https://img.shields.io/badge/MongoDB-8-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
[![Kafka](https://img.shields.io/badge/Apache%20Kafka-Event%20Driven-231F20?logo=apachekafka&logoColor=white)](https://kafka.apache.org/)
[![Redis](https://img.shields.io/badge/Redis-Caching-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](#license)

</div>

---

## 📖 Overview

**CartNova** is a practical, production-oriented e-commerce backend built entirely in Go. It isn't a toy CRUD app — it's a deliberately scoped **microservices system** where every technology earns its place in the architecture, and every line of code is written to be *read and understood*, not generated or compressed into clever one-liners.

The project exists to demonstrate the exact skill set expected of a backend engineer working on distributed systems:

- Designing and building REST + gRPC services that talk to each other correctly
- Modeling real business workflows as **asynchronous domain events** over Kafka
- Solving real concurrency problems (not textbook ones) with goroutines, channels and database transactions
- Keeping the system small enough to fully explain, end to end, in a single sitting

CartNova intentionally ships with **five business services + one API Gateway** — no more. It skips Kubernetes, service mesh and distributed tracing platforms, not because they aren't valuable, but because adding them here would trade clarity for buzzwords.

---

## 🏗️ Architecture

```text
                              Client
                                │
                               REST
                                │
                                ▼
                         ┌───────────────┐
                         │  API Gateway  │
                         └───────┬───────┘
                                 │
          ┌───────────────────────┬───────────────────────┐
          ▼                       ▼                       ▼
   ┌─────────────┐        ┌─────────────┐        ┌─────────────┐
   │ User Service│        │Product Svc  │        │ Order Svc   │
   └──────┬──────┘        └──────┬──────┘        └──────┬──────┘
          │                      │                      │
          ▼                      ▼                      ▼
     PostgreSQL             PostgreSQL                PostgreSQL
                                │                          │
                              Redis                      Kafka
                                                    ┌───────┴───────┐
                                                    ▼               ▼
                                           ┌─────────────┐  ┌───────────────┐
                                           │ Inventory   │  │ Notification  │
                                           │  Service    │  │   Service     │
                                           └──────┬──────┘  └───────┬───────┘
                                                  │                 │
                                              PostgreSQL         MongoDB
                                                                    │
                                                                WebSocket
                                                                    │
                                                                 Client
```

### Services

| # | Service | Responsibility | Storage |
|---|---|---|---|
| 1 | **API Gateway** | Single public entry point — JWT validation, routing, WebSocket proxying | — |
| 2 | **User Service** | Registration, login, profiles, roles (with a seeded local `ADMIN`) | PostgreSQL |
| 3 | **Product Service** | Products & categories; source of truth + Redis-backed read cache | PostgreSQL + Redis |
| 4 | **Order Service** | Owns orders/order items; validates via gRPC, publishes order events | PostgreSQL |
| 5 | **Inventory Service** | Owns stock/reservations; consumes order events concurrently, prevents overselling with row locking | PostgreSQL |
| 6 | **Notification Service** | Consumes order events, stores notifications, pushes updates over WebSocket | MongoDB |

---

## 🔁 Order Workflow

Every order walks through the full event-driven pipeline — synchronous validation first, then asynchronous fan-out:

```text
POST /api/v1/orders
        │
        ▼
   API Gateway
        │
        ▼
   Order Service ──gRPC──► User Service
        │        ──gRPC──► Product Service
        │
        │  create order transaction
        ▼
     PostgreSQL
        │
        │  OrderCreated
        ▼
      Kafka
      ╱    ╲
     ▼      ▼
Inventory  Notification ──► MongoDB
    │
    │  InventoryReserved / InventoryReservationFailed
    ▼
  Kafka
    │
    ▼
Order Service ──► CONFIRMED or CANCELLED
    │
    ▼
  Kafka
    │
    ▼
Notification Service ──► WebSocket ──► Client
```

1. The gateway authenticates the request and forwards it to the **Order Service**.
2. Order Service validates the user and products **synchronously over gRPC** — no point creating an order for a product or user that doesn't exist.
3. The order is written inside a PostgreSQL transaction, then an `OrderCreated` event is published to Kafka.
4. **Inventory** and **Notification** both consume the event independently and in parallel.
5. Inventory reserves stock — success or failure is published back to Kafka as its own event.
6. Order Service consumes that result and transitions the order to `CONFIRMED` or `CANCELLED`, emitting a final status event.
7. Notification Service picks that up and pushes it live to the client over a **WebSocket**.

---

## ⚡ Concurrency Design

Concurrency here isn't decorative — it exists because the Inventory Service has a real correctness problem to solve: **multiple order events arriving for the same product, potentially across multiple service instances, must never oversell stock.**

- Kafka partitions allow multiple consumers to process events in parallel.
- Each consumer runs a pool of goroutines around a shared jobs channel for controlled concurrent processing.
- **Correctness is owned by PostgreSQL, not by Go.** A `sync.Mutex` only protects memory inside a single process — it does nothing once you run more than one instance of the service. So the inventory transaction locks the product row with `FOR UPDATE`, checks available stock, updates the reservation, and commits — all atomically, at the database level.

Concepts that would feel artificial if forced into a business feature are instead demonstrated in isolation under `examples/concurrency`, so each one can be studied on its own before tracing it through the real distributed workflow.

**Concurrency topics covered:**

<table>
<tr>
<td valign="top">

- Goroutines
- Unbuffered channels
- Buffered channels
- Channel closing & `range`
- `select`
- `sync.WaitGroup`

</td>
<td valign="top">

- `sync.Mutex` / `sync.RWMutex`
- Worker pools
- Producer/consumer
- Fan-in / fan-out
- Pipelines

</td>
<td valign="top">

- `context.Context`
- Cancellation & timeouts
- Race conditions & deadlocks
- Goroutine leaks
- `go test -race ./...`

</td>
</tr>
</table>

---

## 🧰 Tech Stack

| Layer | Technology | Why it's here |
|---|---|---|
| Language | **Go 1.23** | Fast, statically typed, first-class concurrency |
| Public API | **REST** (`net/http`) | Simple, well-understood public contract |
| Internal RPC | **gRPC** | Low-overhead synchronous service-to-service calls |
| Messaging | **Apache Kafka** | Asynchronous domain events, decoupled services |
| Cache | **Redis** | Product read-path caching |
| Relational DB | **PostgreSQL 17** | Transactional business data, row-level locking |
| Document DB | **MongoDB 8** | Notification documents |
| Real-time | **WebSockets** | Live order status delivery to the client |
| Auth | **JWT + bcrypt** | Stateless authentication and password security |
| Containers | **Docker + Docker Compose** | Reproducible local infrastructure |
| Cloud | **AWS** (deployment docs) | Real-world deployment target |
| Testing | **Go testing + `-race`** | Correctness and data-race detection |

---

## 📁 Project Structure

```text
cartnova/
├── cmd/                     # Entry point for each service
│   ├── gateway/
│   ├── user-service/
│   ├── product-service/
│   ├── order-service/
│   ├── inventory-service/
│   └── notification-service/
│
├── internal/                # Application code, per domain
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
├── migrations/              # Per-service SQL migrations
├── proto/                   # gRPC service definitions
├── examples/concurrency/    # Standalone concurrency demos
├── docs/                    # Architecture, API, events, concurrency docs
├── deployments/
├── scripts/                 # Smoke test, utility scripts
├── web/                     # Minimal client for WebSocket demo
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## 🌐 Local Service Ports

| Component | HTTP | gRPC |
|---|---:|---:|
| API Gateway | `8080` | — |
| User Service | `8081` | `9091` |
| Product Service | `8082` | `9092` |
| Order Service | `8083` | — |
| Inventory Service | `8084` | — |
| Notification Service | `8085` | — |
| Kafka | `29092` | — |
| PostgreSQL | `5432` | — |
| Redis | `6379` | — |
| MongoDB | `27017` | — |

---

## 🚀 Getting Started

### Prerequisites

- Go 1.23+
- Docker Desktop + Docker Compose
- Git, `curl`
- Python 3 (used only for JSON extraction in the smoke test)

### Option A — Run everything with Docker

```bash
# 1. Copy the environment template
cp .env.example .env

# 2. Set a real JWT_SECRET in .env

# 3. Build and start every service + infrastructure
docker compose up --build -d

# 4. Confirm the gateway is healthy
curl http://localhost:8080/health

# 5. Run the end-to-end smoke test
make smoke
```

To wipe all local databases and Kafka data and start clean:

```bash
docker compose down -v
docker compose up --build -d
```

> ⚠️ `-v` deletes local volumes permanently — never run this against a real environment.

### Option B — Run infrastructure in Docker, services locally

Useful when actively developing or debugging a single service.

```bash
# Start only the infrastructure
docker compose up -d postgres redis mongodb kafka kafka-init

# Install Go dependencies
go mod tidy

# Run each service in its own terminal
go run ./cmd/user-service
go run ./cmd/product-service
go run ./cmd/order-service
go run ./cmd/inventory-service
go run ./cmd/notification-service
go run ./cmd/gateway
```

Default local ports and connection strings are in `.env.example`.

---

## 🧪 Testing

```bash
# Run the full test suite
go test ./...

# Run with Go's race detector — critical for the concurrency-heavy services
go test -race ./...

# Format the codebase
make fmt
```

---

## 🔑 Development Credentials

> These are **local development defaults only**. Change or remove them before any real deployment.

**PostgreSQL**
```text
user: postgres
password: root
```

**MongoDB**
```text
user: root
password: root
```

**Seeded CartNova admin account**
```text
email: admin@cartnova.local
password: password
```

---

## 📚 Documentation

| Doc | Covers |
|---|---|
| [`docs/architecture.md`](docs/architecture.md) | Full system design and service boundaries |
| [`docs/api.md`](docs/api.md) | REST API reference |
| [`docs/events.md`](docs/events.md) | Kafka event schemas and topics |
| [`docs/concurrency.md`](docs/concurrency.md) | Deep dive into the concurrency model |

---




## 🎯 Design Principle

Every piece of infrastructure in CartNova is here for a specific reason, not for the sake of a longer tech-stack list:

- **REST** — public API contract
- **gRPC** — low-overhead synchronous internal calls
- **Kafka** — asynchronous domain events between services
- **Redis** — product read-path caching
- **PostgreSQL** — transactional business data
- **MongoDB** — notification documents
- **WebSockets** — real-time delivery to the client
- **Goroutines/channels** — controlled, correct concurrent processing
- **Docker** — reproducible local infrastructure
- **AWS** — the real deployment target

Kubernetes, service mesh and distributed tracing are deliberately left out — they solve problems this project doesn't have yet.

---

## 📄 License

This project is licensed under the MIT License.

---

<div align="center">

Built by [Shivam Nayak](https://github.com/ShivamNayak-dev)

</div>
