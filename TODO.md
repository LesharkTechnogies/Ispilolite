# Production TODO

This checklist reflects the current repository after the location-learning, job, quotation, package-catalog, monitoring, queue, and authentication work.

## Completed

- [x] Phone and case-insensitive username uniqueness with PostgreSQL constraints.
- [x] OTP registration, access tokens, refresh tokens, and database-backed refresh sessions.
- [x] Location learning with distinct-user submissions, county hierarchy, popularity, and verification.
- [x] County/town/village provider search and locality-aware recommendations.
- [x] ISP coverage management and coverage recommendations.
- [x] Customer direct requests and broadcast jobs.
- [x] Multi-provider job applications and transactional customer assignment.
- [x] Job availability, soft deletion, status transitions, and notification events.
- [x] Quotation finalization with server-side calculations, tax snapshots, public ILO codes, and persistence.
- [x] Quotation units, custom units, tax rates, item discounts, transport, VAT, and payment snapshots.
- [x] Prometheus HTTP, database, search, business, and RabbitMQ metrics.
- [x] RabbitMQ durable exchanges, queues, confirmations, consumers, retries through DLQs, and notification worker.
- [x] Normalized ISP package catalog with reusable speed/data units.
- [x] ISP PPPoE and Hotspot packages.
- [x] Package speed filtering, price-range filtering, cheapest sorting, and county-specific effective prices.
- [x] ISP profile sorting by cheapest matching package with county/category/price/speed filters.
- [x] Package capacity selling with unlimited/capped capacity fields.
- [x] ISP package creation, update, county-price update, profile package loading, and public package discovery.
- [x] Shared PostgreSQL search adapter now delegates to the maintained production search implementation.
- [x] ISP and technician review persistence, listing, creation, and uniqueness checks.
- [x] Baseline request logging and in-process rate limiting middleware.
- [x] Redis session repository for cached user/device sessions.
- [ x] Add database migration version tracking instead of executing every SQL file on startup.
- [ x] Add database backup, restore, retention, and disaster-recovery procedures.
- [x] Add PostgreSQL connection health, lock, slow-query, and replication-lag dashboards.
- [x] Add distributed tracing with OpenTelemetry and Jaeger.
- [x] Add RabbitMQ connection recovery, consumer restart supervision, and DLQ replay tooling.
- [x] Add Talksasa SMS provider for OTP, hashed transactional messages, and admin-targeted SMS.
- [ ] Add real FCM, email, and push provider adapters behind the notification webhook boundary.
 - [x] Add package versioning so historical customer subscriptions retain their original commercial terms.
 - [x] Add review aggregation updates to ISP/technician rating and review-count columns.
- [x] Add review moderation, abuse reporting, and admin approval workflows.
- [x] Implement technician profile/posts repositories and public portfolio endpoints.
- [x] Add location fuzzy aliases and administrative-boundary validation.
- [x] Add Elasticsearch index lifecycle, reindex, and mapping migration jobs.
- [x] Add API pagination metadata and cursor pagination for high-volume search lists.
- [x] Backfill legacy `isp_packages.speed` and `isp_packages.price` into normalized speed units/base prices.
- [x] Add package archive/delete endpoints and prevent changes to packages referenced by active subscriptions.
- [x] Add package availability/capacity reservation and subscription lifecycle enforcement.
- [x] Add WebSocket or SSE delivery for notification updates.
- [x] Add PDF rendering/storage and quotation download endpoints through the authorized Go API.

Task 3 — add a Postgres health collector emitting those 4 metrics (query pg_stat_activity for waiting locks, long-running queries for slow-query count, pg_last_xact_replay_timestamp for replica lag, ping for up). Small, additive, low-risk, and it makes the existing dashboard light up. [x]
Task 7 — auto-reconnect, consumer restart supervision, and a DLQ replay command. Medium; modifies a currently-working queue path. [x]
Task 6 — OpenTelemetry SDK + Jaeger exporter, tracer bootstrap in cmd/auth/main.go, HTTP/DB/queue span instrumentation. Largest, and it needs new Go modules fetched + go.sum regenerated (the Deployment section of the TODO lists that as still-pending, so the toolchain/network state here is uncertain). [x]
## High Priority Production Work

- [ ] Add integration tests against PostgreSQL for all migrations and transaction paths.
- [ ] Add unit tests for quotation calculations, discounts, VAT inclusive/exclusive, package filters, and job assignment races.
- [ ] Add API contract tests for Flutter payloads and error responses.


- [ ] Add idempotency keys for customer requests, quotation finalization, applications, and payments.
- [ ] Add API authentication key rotation and refresh-token revocation for all sessions on logout/password change.
- [ ] Add admin APIs for tax rates, system units, business profiles, package moderation, and audit logs.

- [ ] Add quotation payment verification and watermark entitlement checks.
- [ ] Add ISP package inventory/capacity enforcement when a package is sold.


## Medium Priority


- [ ] Add Grafana dashboards and Prometheus alert rules for p95 latency, error rate, DB saturation, queue depth, and search fallback rate.
- [ ] Add OpenAPI generation and CI schema compatibility checks.

## Deployment And Security

- [ ] Regenerate `go.sum` with the installed Go toolchain and run `go test ./...`.
- [ ] Run `go vet ./...`, static analysis, race tests, and vulnerability scanning in CI.
- [ ] Remove development JWT fallback secrets from production startup.
- [ ] Store RabbitMQ, database, Redis, JWT, SMS, email, and webhook secrets in a secret manager.
- [ ] Configure TLS for PostgreSQL, Redis, RabbitMQ, Elasticsearch, and public APIs.
- [ ] Restrict `/metrics` to the monitoring network or protect it with network policy.
- [ ] Configure RabbitMQ users, vhosts, permissions, quorum queues, and multi-node deployment.
- [ ] Configure Prometheus 30-day retention and Grafana persistent storage.
- [ ] Add Kubernetes readiness/liveness probes and resource limits for every worker.

