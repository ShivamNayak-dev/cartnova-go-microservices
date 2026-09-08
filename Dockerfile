FROM golang:1.23-alpine AS builder

ARG SERVICE
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM alpine:3.20

RUN addgroup -S cartnova && adduser -S cartnova -G cartnova
WORKDIR /app

COPY --from=builder /out/service /app/service
COPY migrations /app/migrations

USER cartnova
EXPOSE 8080

ENTRYPOINT ["/app/service"]
