# Data Consistency Demo — Microservice Pattern

## Overview

This repository is a demonstration of microservice patterns for maintaining data consistency across services. It includes a simple orchestrator, several service implementations, a projector, and a frontend UI for exploration.

## Architecture

- **external-orchestrator**: orchestration / API layer coordinating workflows.
- **inventory-service**: manages inventory and publishes events.
- **payment-service**: handles payments and publishes events.
- **projector-service**: reads events and projects read-models.
- **scripts/init-mongo.js**: database initialization script used by the compose stack.
- **config/prometheus.yml**: Prometheus configuration for monitoring.

Services communicate via events and use MongoDB for persistence in this demo.

## Prerequisites

- Docker & Docker Compose
- (Optional) Go toolchain for running services locally
- (Optional) Node.js & npm/yarn for frontend development

## Quick start (docker-compose)

1. From the repository root, build and start all services:

```bash
docker-compose up --build
```

2. Wait for services to initialize. The `scripts/init-mongo.js` script will seed MongoDB where applicable.

3. Open the frontend. If the compose stack exposes a web UI, check the compose output for the forwarded port.

## Run services individually (development)

- Orchestrator / services (example):

```bash
# build and run a Go service from its folder
cd external-orchestrator
go run ./main.go
```
Adjust service start commands to your local Go environment and module paths.

## Testing

- Trigger the example workflow by POSTing to the `external-orchestrator` API (adjust host/port/route as needed):

```bash
curl --location 'localhost:8080/orders' \
--header 'Content-Type: application/json' \
--data '{
    "order_id": "ORD-001",
    "product_id": "iphone-15",
    "qty": 1,
    "amount": 10000
}'
```

- Monitor workflow execution in the Temporal Web UI. When running via Docker Compose the UI is often exposed at `http://localhost:8000` (check `docker-compose.yml` for the actual port mapping).

- Connect to the services' database (MongoDB). Example connection strings and commands:

```bash
# connect with a URI (use in MongoDB Compass or other clients)
mongodb://localhost:27017/?directConnection=true
```

The `scripts/init-mongo.js` script seeds initial data when the compose stack starts; inspect or run it manually if you need custom seed data.

## Monitoring

Prometheus configuration is available at `config/prometheus.yml`. When running with Docker Compose, Prometheus can scrape instrumented services if the compose file maps the scrape targets.

## Scripts

- `scripts/init-mongo.js` — seeds MongoDB used by services in the demo.

## MongoDB — Collections & Indexes

- **Database:** `shop_db`

- **events** (Event Store)
    - Fields: `stream_id`, `type`, `qty`, `version`, `timestamp`.
    - Indexes: unique index on `{stream_id: 1, version: 1}` to enforce optimistic locking/idempotency. Created by `scripts/init-mongo.js`.
    - Notes: events are appended and read in timestamp/version order. Queries often filter by `stream_id`.

- **products_view** (Read Model)
    - Fields: `product_id`, `available_stock`, `last_version`.
    - Indexes: unique index on `{product_id: 1}` for fast lookups and idempotency. Created by `scripts/init-mongo.js`.

- **checkpoints** (Projector state)
    - Fields: `_id`, `resume_token`.
    - Usage: projector saves its resume token here (document key is the projector name).

- **payment_events**
    - Fields: `_id`, `order_id`, `amount`, `type`, `status`, `timestamp`.
    - Indexes: none created by the seed script; recommended indexes: `{order_id: 1}` for stream/lookup queries and `{timestamp: -1}` for recent-event queries.

The `scripts/init-mongo.js` script creates the `events` and `products_view` collections and their core indexes; review or extend it if you need additional indexes for production workloads.

## Contributing

Contributions and improvements are welcome. Open an issue describing the change or a PR with a clear description and small, focused commits.

## License

This demo is provided for learning and experimentation. No license file included; treat as MIT-like for personal/educational use unless otherwise specified.

---

If you want, I can add specific run commands and ports by reading `docker-compose.yml` and service entrypoints next.
