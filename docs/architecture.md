# CartNova Architecture

## Services

| Service | Responsibility | Storage |
|---|---|---|
| API Gateway | Public entry point, routing, JWT validation | None |
| User Service | Registration, login, profiles | PostgreSQL |
| Product Service | Products, categories, search/filter, caching | PostgreSQL + Redis |
| Order Service | Orders and order lifecycle | PostgreSQL |
| Inventory Service | Stock and reservations | PostgreSQL |
| Notification Service | Notification history and real-time updates | MongoDB |

## Communication rules

### REST

Client to API Gateway.

### gRPC

Selected synchronous internal calls. Order Service calls User and Product services through gRPC.

### Kafka

Asynchronous domain events. Order and Inventory communicate through events, and Notification reacts to order events.

### WebSockets

Notification Service pushes order updates to connected clients.

## Database ownership

Each service owns its data. There is no shared application-level database schema.

## Caching

Product details use a cache-aside Redis strategy:

```text
read product
    |
    v
 Redis ---- hit ---> response
    |
   miss
    v
PostgreSQL
    |
    v
 Redis
```

## Security

The gateway validates JWTs. Services also validate JWTs for direct/local access. ADMIN-only operations are checked in the owning service.

## Known trade-offs

CartNova intentionally avoids a full distributed transaction coordinator. The order workflow is eventually consistent across services and relies on transactions within each service plus idempotent event handling.

For a larger production system, an outbox pattern would be a natural next improvement to guarantee database changes and event publication are durably coordinated.
