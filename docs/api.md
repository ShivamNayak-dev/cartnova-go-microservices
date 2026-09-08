# CartNova API

The public API is exposed through the API Gateway on `http://localhost:8080`.

## Authentication

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
```

Login returns a JWT. Send it as:

```http
Authorization: Bearer <token>
```

The seeded local admin is:

```text
email: admin@cartnova.local
password: password
```

Change this before any non-local use.

## Users

```http
GET /api/v1/users/{id}
PUT /api/v1/users/{id}
```

Users can update their own profile. ADMIN can update any profile.

## Categories

```http
POST   /api/v1/categories       # ADMIN
GET    /api/v1/categories
GET    /api/v1/categories/{id}
PUT    /api/v1/categories/{id}  # ADMIN
DELETE /api/v1/categories/{id}  # ADMIN
```

## Products

```http
GET    /api/v1/products
GET    /api/v1/products/{id}
POST   /api/v1/products          # ADMIN
PUT    /api/v1/products/{id}     # ADMIN
DELETE /api/v1/products/{id}     # ADMIN
```

Filtering examples:

```text
GET /api/v1/products?q=keyboard
GET /api/v1/products?category_id=1
```

## Orders

```http
POST /api/v1/orders
GET  /api/v1/orders/mine
GET  /api/v1/orders/{id}
PUT  /api/v1/orders/{id}/status  # ADMIN
```

Create order example:

```json
{
  "items": [
    {
      "product_id": 1,
      "quantity": 2
    }
  ]
}
```

Valid order statuses:

```text
CREATED
CONFIRMED
PROCESSING
SHIPPED
DELIVERED
CANCELLED
```

## Inventory

```http
GET  /api/v1/inventory/{productID}
POST /api/v1/inventory             # ADMIN
```

Adding inventory:

```json
{
  "product_id": 1,
  "quantity": 10
}
```

## Notifications

```http
GET /api/v1/notifications
```

## WebSocket

```text
ws://localhost:8080/ws/notifications?token=<JWT>
```

A browser client can use the query-token form because browser WebSocket APIs do not provide a general custom Authorization-header API.
