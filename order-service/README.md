# Order Service

Part of AP2 Assignment 1 — Clean Architecture Microservices.

This service manages customer orders and communicates with the Payment Service via REST.

## Architecture

- **Domain**: `internal/domain/` — Order entity, domain errors
- **Use Case**: `internal/usecase/` — Business logic, ports (interfaces)
- **Repository**: `internal/repository/` — PostgreSQL implementation
- **Transport**: `internal/transport/http/` — Gin handlers, DTOs, router
- **App**: `internal/app/` — Configuration, DB connection, Payment HTTP client

## Endpoints

- `POST /orders` — Create order (calls Payment Service)
- `GET /orders/:id` — Get order by ID
- `PATCH /orders/:id/cancel` — Cancel order (only Pending)