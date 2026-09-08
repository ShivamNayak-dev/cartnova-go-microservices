# Data Consistency Notes

## Inventory

Inventory is the strongest consistency boundary in CartNova.

A reservation is handled inside a PostgreSQL transaction. `SELECT ... FOR UPDATE` serializes competing updates to the same product row.

## Orders and inventory

The order is created first and starts in `CREATED`. Inventory reservation happens asynchronously. Therefore the system is intentionally eventually consistent between the two services.

Possible outcome:

```text
OrderCreated
    |
    v
Inventory reservation succeeds
    |
    v
OrderConfirmed
```

or:

```text
OrderCreated
    |
    v
Inventory reservation fails
    |
    v
OrderCancelled
```

## Why not a distributed transaction?

Distributed transactions would add complexity and reduce the educational value of the project. CartNova instead demonstrates local ACID transactions plus asynchronous event coordination.
