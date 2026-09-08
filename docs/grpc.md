# gRPC in CartNova

CartNova uses gRPC for selected internal synchronous calls.

The service contracts are documented in:

- `proto/product.proto`
- `proto/user.proto`

The current implementation uses `google.protobuf.Struct` messages so the repository remains self-contained without requiring generated `protoc` artifacts during development. The service descriptors are registered in `internal/grpcapi`.

## Product lookup

Order Service sends:

```json
{
  "id": 1
}
```

to Product Service and receives product information such as price and status.

## User lookup

Order Service sends:

```json
{
  "id": 101
}
```

to User Service to verify that the authenticated user exists.

## Why gRPC here?

The calls are internal, synchronous operations where Order Service needs an immediate answer before creating the order. Kafka would be inappropriate for this specific validation step because the caller needs the result immediately.
