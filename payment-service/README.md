# Payment Service

Part of AP2 Assignment 1 — Clean Architecture Microservices.

This service processes payments and validates transaction limits.

## Architecture

- **Domain**: `internal/domain/` — Payment entity, domain errors, constants
- **Use Case**: `internal/usecase/` — Business logic, ports (interfaces)
- **Repository**: `internal/repository/` — PostgreSQL implementation
- **Transport**: `internal/transport/http/` — Gin handlers, DTOs, router
- **App**: `internal/app/` — Configuration, DB connection

## Endpoints

- `POST /payments` — Authorize payment (limit check: max 100000)
- `GET /payments/:order_id` — Get payment by order ID