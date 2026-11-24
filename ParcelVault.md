# ParcelVault

**Track your parcels. Not you.**

Technical specification • Draft v0.1 • 24 November 2026

## 1. Vision and scope

ParcelVault is an open-source, self-hosted, API-first parcel and shopping order management platform. A Go server owns the data and performs background tracking; interchangeable clients read its API. Personal use is the first target, with public distribution and community clients as longer-term goals.

Requirements: no telemetry, advertising, mandatory cloud account or location tracking; local AI optional; native iPhone/iPad client; Home Assistant integration; automatic order discovery; cached courier data; secure multi-user access; portable data and headless operation. Internet access remains necessary for external courier and email services. No live integration or implementation is claimed by this document.

## 2. Architecture

``` mermaid
flowchart TD
  IOS[iOS client] --> API
  WEB[Web UI] --> API
  THIRD[Community clients] --> API
  HA[Home Assistant] --> API
  subgraph Host[Self-hosted Go server]
    API[Versioned REST API] --> AUTH[Auth and authorisation]
    AUTH --> SVC[Domain services]
    SVC --> DB[(SQLite or PostgreSQL)]
    DB --> JOB[Persistent job queue]
    JOB --> WORK[Bounded workers]
    WORK --> DB
    DB --> OUT[Transactional outbox]
    OUT --> DIS[Event dispatcher]
  end
  WORK --> C[Courier APIs]
  WORK --> E[Email and retailer APIs]
  WORK --> AI[Local AI endpoint]
  DIS --> MQTT[MQTT broker]
  MQTT --> HA
  DIS --> WEBHOOK[Webhooks and notifications]
  DIS --> STREAM[SSE clients]
```

Use a modular monolith, not microservices. Domain logic depends on interfaces; adapters handle external protocols. The API and workers can initially run in one process. A separate worker mode may be added for larger deployments. Clients never directly access the database or courier credentials.

### Proposed stack

  ---------------------------------------------------------------------------------------------------------
  Area                                Choice
  ----------------------------------- ---------------------------------------------------------------------
  Backend                             Go, net/http and optionally Chi

  API                                 REST JSON, OpenAPI 3.1, SSE; WebSocket if bidirectional needs arise

  IDs                                 google/uuid NewV7

  Authentication                      golang-jwt/jwt/v5, Ed25519, Argon2id

  Database                            SQLite default, PostgreSQL optional

  Queries and migrations              sqlc, Goose; dialect-specific SQL

  Drivers                             modernc.org/sqlite, pgx

  Scheduling                          Database-backed leases and bounded goroutines

  Rate limiting                       golang.org/x/time/rate plus persistent quotas

  Logs                                log/slog, local metrics, optional OpenTelemetry export

  Web                                 React/TypeScript, optionally embedded with go:embed

  Mobile                              SwiftUI, Keychain; Android later

  Deployment                          Docker Compose, AMD64/ARM64, standalone binary

  Licence                             AGPL-3.0 proposed; confirm before publication
  ---------------------------------------------------------------------------------------------------------

## 3. Repository layout

``` text
parcelvault/
  cmd/parcelvault/
  internal/
    api/ auth/ config/ database/ users/ households/
    parcels/ orders/ tracking/ scheduler/ couriers/
    integrations/ ai/ events/ mqtt/ webhooks/
    notifications/ security/ audit/
  api/openapi.yaml
  migrations/sqlite/
  migrations/postgres/
  queries/
  web/
  clients/ios/
  integrations/home-assistant/
  docs/architecture/
  docs/adr/
  docs/security/
  docs/diagrams/
  Dockerfile
  compose.yaml
  SECURITY.md
  CONTRIBUTING.md
  README.md
```

## 4. Data model

``` mermaid
erDiagram
  USER ||--o{ SESSION : owns
  USER ||--o{ API_TOKEN : owns
  USER ||--o{ MEMBERSHIP : has
  HOUSEHOLD ||--o{ MEMBERSHIP : contains
  USER ||--o{ ORDER : owns
  USER ||--o{ PARCEL : owns
  USER ||--o{ INTEGRATION : connects
  ORDER ||--o{ ORDER_ITEM : contains
  ORDER_ITEM ||--o{ SHIPMENT_ITEM : allocated
  SHIPMENT ||--o{ SHIPMENT_ITEM : contains
  SHIPMENT o|--o{ PARCEL : includes
  PARCEL ||--o{ TRACKING_EVENT : records
  PARCEL ||--o{ REFRESH_JOB : schedules
  COURIER ||--o{ PARCEL : transports
  ORDER ||--o{ RETURN : has
  RETURN ||--o{ RETURN_ITEM : contains
  RETURN o|--o{ PARCEL : uses
  ORDER ||--o{ REFUND : receives
```

  -------------------------------------------------------------------------------------------------------------------------------------------
  Table                               Key fields beyond ID
  ----------------------------------- -------------------------------------------------------------------------------------------------------
  users                               username, password_hash, created_at

  households / memberships            household_id, user_id, role

  sessions                            user_id, expiry, revoked_at, token_family

  refresh_tokens / api_tokens         token_hash, scopes, expiry, revocation

  orders                              owner_id, retailer, external_reference, ordered_at, currency

  order_items                         order_id, name, quantity, amount_minor

  shipments / shipment_items          ownership, item allocation, quantity

  parcels                             owner_id, shipment_id nullable, courier_id, tracking_number, status, last_success_at, next_refresh_at

  tracking_events                     parcel_id, provider_event_id, event_at, received_at, raw_status, normalised_status

  integrations                        owner_id, provider, encrypted credentials, sync cursor

  refresh_jobs                        parcel_id, due_at, lease_until, lease_token, attempts, state

  api_usage                           provider credential scope, window, consumed quota

  outbox / deliveries                 event_id, payload, schema_version, destination, retry state

  returns / return_items / refunds    order links, quantities, deadlines, amounts, state

  audit_logs                          actor, action, resource, timestamp, redacted metadata
  -------------------------------------------------------------------------------------------------------------------------------------------

Use UUIDv7 for resource IDs and explicit UTC timestamps for creation, updates and actual courier events. UUIDv7 exposes generation time and is not a secret. Use random opaque secrets for tokens. Use integer minor units and ISO currency codes for money. Index ownership and pagination keys, due jobs and tracking history. Enforce foreign keys, unique import identifiers and event deduplication. Use transactions for state changes and outbox insertion. Household sharing is explicit, not automatic. Validate cross-owner relationships. SQLite uses WAL and a conservative writer strategy; PostgreSQL supports concurrent workers and optional row-level security. Test both dialects independently.

## 5. API contract

All private endpoints require authentication and resource-level authorisation. UUIDv7 appears in resource URLs, never tracking numbers or credentials. Version breaking changes under a new major path. Publish OpenAPI and generate clients where useful.

  Method             Path (prefix /api/v1)   Purpose
  ------------------ ----------------------- ---------------------------
  POST               /auth/login             Authenticate
  POST               /auth/refresh           Rotate refresh token
  POST               /auth/logout            Revoke session
  GET/POST           /parcels                List/create
  GET/PATCH/DELETE   /parcels/{id}           Read/update/delete
  POST               /parcels/{id}/refresh   Queue refresh; 202
  GET                /parcels/{id}/events    Tracking history
  GET/POST           /orders                 List/create orders
  GET/PATCH/DELETE   /orders/{id}            Manage order
  GET                /couriers               Provider capabilities
  GET                /statistics             Authorised aggregates
  GET/POST           /integrations           Manage connections
  GET                /returns                Returns
  POST               /ai/chat                Read-only local assistant
  GET                /events                 Authorised SSE stream
  GET                /health                 Minimal liveness only

Use opaque cursor pagination (created_at and ID tie-breaker), bounded page sizes, RFC 9457 problem details, request size limits, idempotency keys for retried creates, and ETags where appropriate. Return 401 for missing/invalid auth, 403 for insufficient scopes, 404 for inaccessible individual resources where appropriate, 429 with Retry-After for rate limits. Private responses must not be cached by shared proxies.

``` json
{
  "id": "0193a2b4-c5d6-7abc-8def-123456789abc",
  "courier": "royal_mail",
  "status": "out_for_delivery",
  "cache": {
    "last_successful_refresh": "2026-11-24T09:00:00Z",
    "next_refresh": "2026-11-24T09:15:00Z",
    "is_stale": false
  }
}
```

## 6. Authentication and security

``` mermaid
sequenceDiagram
  participant C as Client
  participant A as Auth service
  participant D as Database
  participant P as Parcel API
  C->>A: Authenticate
  A->>D: Verify account and create session
  A-->>C: Short-lived JWT and opaque refresh token
  C->>P: Bearer JWT
  P->>P: Verify signature, issuer, audience, expiry, scopes
  P->>D: Check session and resource ownership
  D-->>P: Authorised data
  P-->>C: JSON
  C->>A: Refresh token
  A->>D: Atomically rotate token
  A-->>C: New token pair
```

Proposed defaults: 15-minute access JWT, 30-day refresh session with rotation and reuse detection, per-installation Ed25519 keys and key rotation. Validate algorithm allowlist, issuer, audience, expiry and session revocation. Hash opaque tokens at rest; encrypt recoverable integration secrets with authenticated encryption and a separately protected key. Never put personal data in JWTs. Use Argon2id for passwords, rate-limit login, no default password, one-time local bootstrap. Support TOTP, recovery codes, passkeys and OIDC in later phases. Native and third-party OAuth clients use authorisation code with PKCE and system browser; use established libraries or identity provider rather than inventing OAuth. Web sessions use Secure HttpOnly SameSite cookies and CSRF protection. iOS stores tokens in Keychain. Home Assistant uses narrowly scoped revocable service tokens. Admin, member and viewer roles; ownership checks on every resource and stream. Host administrators can access underlying storage; do not promise isolation from root.

  ------------------------------------------------------------------------------------------------------------------------------------
  Threat                              Control
  ----------------------------------- ------------------------------------------------------------------------------------------------
  Guessed UUID or endpoint            Deny by default; ownership and scope checks

  Stolen token                        Short lifetime, revocation, rotation, reuse detection

  Injection                           Parameterised SQL, output escaping, validation

  SSRF                                Restrict destinations, redirects and DNS resolution; explicitly configured local AI exceptions

  Malicious email / AI                No tool execution, remote images or automatic link following

  Credential theft                    Encrypted secrets, separate key, redacted logs

  Abuse / exhaustion                  Rate limits, bounded queues, timeouts, body limits

  Supply chain                        Dependency scanning, SBOM, signed release images

  Data loss                           Tested encrypted backups and restore
  ------------------------------------------------------------------------------------------------------------------------------------

TLS required outside development; private VPN preferred over public exposure. Restrict CORS and trusted proxy headers. Run container as non-root with minimal privileges. Local audit logs have retention controls and no secrets. Document vulnerability reporting in SECURITY.md. Security tests cover cross-user access, token replay, refresh races, webhook validation and SSRF.

## 7. Tracking engine and caching

``` mermaid
flowchart TD
  A[Create parcel] --> B[Persist refresh job]
  B --> C{Due and quota available?}
  C -->|No| D[Wait or reschedule]
  D --> C
  C -->|Yes| E[Acquire job lease]
  E --> F[Fetch courier status]
  F --> G{Success?}
  G -->|No| H[Preserve cached data and back off]
  H --> B
  G -->|Yes| I[Deduplicate and save events plus outbox]
  I --> J{Terminal status?}
  J -->|No| B
  J -->|Yes| K[Stop routine polling]
```

  Status                             Proposed polling interval
  ---------------------------------- ---------------------------------
  Awaiting dispatch                  6 hours, primarily email sync
  Label created                      2 hours
  In transit                         60 minutes
  Arriving today                     30 minutes
  Out for delivery                   15 minutes
  Delayed                            2 hours
  Delivered / cancelled / returned   Stop routine polling
  Failure                            Exponential backoff with jitter

Intervals are configurable proposals, not verified courier allowances. Provider limits always override them. Respect Retry-After, persist quota consumption across restarts, batch where supported, deduplicate concurrent refreshes, apply manual refresh cooldowns and use signed provider webhooks where available. Use job leases with fencing tokens to reject stale worker writes; recover expired leases. Display last successful refresh separately from last attempt. Never erase cached status on upstream failure. Retain history and permit authorised reconciliation of terminal states. Clients read database only; refresh requests enqueue work rather than synchronously calling couriers.

``` go
type CourierProvider interface {
    ID() string
    Capabilities() ProviderCapabilities
    Track(ctx context.Context, req TrackingRequest) (TrackingResult, error)
}
```

Target Royal Mail, Parcelforce, DPD, UPS, Evri, DHL, Yodel and FedEx. Direct API access, pricing and permitted caching must be verified per provider. Optional aggregator adapter and official tracking links are fallbacks. No bypassing access restrictions. Start with a fake provider for tests and one authorised live provider for validation.

``` mermaid
stateDiagram-v2
  [*] --> LabelCreated
  LabelCreated --> InTransit
  InTransit --> OutForDelivery
  InTransit --> Delayed
  Delayed --> InTransit
  OutForDelivery --> Delivered
  OutForDelivery --> DeliveryAttempted
  DeliveryAttempted --> AwaitingCollection
  DeliveryAttempted --> OutForDelivery
  AwaitingCollection --> Delivered
  InTransit --> ReturningToSender
  ReturningToSender --> Returned
  LabelCreated --> Cancelled
```

This is illustrative, not a strict transition whitelist: couriers can skip stages, correct events and deliver updates out of order.

## 8. Email, retailer and local AI imports

``` mermaid
flowchart TD
  A[Selected mailbox or retailer API] --> B[Sanitise and filter]
  B --> C[Deterministic parser]
  C --> D{Recognised?}
  D -->|No| E{Local AI enabled?}
  E -->|Yes| F[Structured extraction]
  E -->|No| G[Manual review]
  D -->|Yes| H[Validate fields]
  F --> H
  H --> I{Reliable and matched?}
  I -->|No| G
  I -->|Yes| J[Upsert order and shipment]
  J --> K[Register tracking job if available]
```

Support Gmail and Outlook OAuth where approved, iCloud and other IMAP providers, and a dedicated forwarding mailbox. Selected folders minimise processing but may not restrict the actual OAuth grant; a dedicated mailbox provides stronger isolation. Read-only access, incremental sync cursors, bounded backfill, deduplication by message and retailer order references. Do not load remote images or automatically follow links. Retain only necessary fields by default. eBay buyer API access is restricted; Amazon, Shopify and Etsy email parsing is the initial fallback. Confirm provider approval and terms before public release.

Local AI adapters: Ollama first, then LM Studio, LocalAI, llama.cpp and compatible endpoints. No cloud fallback. Use deterministic parsers first, strict structured output schemas, evidence validation and review for uncertain matches. Treat email text as untrusted data, never instructions. AI has no shell, credentials, unrestricted network or direct database writes. Read-only assistant queries use scoped services and minimise data sent to the model. Support model selection, timeouts, resource limits and no-AI mode. Verify that configured AI endpoints are actually local or explicitly trusted; compatibility alone does not guarantee privacy.

## 9. Events, Home Assistant and notifications

``` mermaid
sequenceDiagram
  participant W as Worker
  participant D as Database
  participant O as Outbox dispatcher
  participant M as MQTT broker
  participant H as Home Assistant
  W->>D: Atomic parcel update and outbox insert
  O->>D: Claim event
  O->>M: Publish scoped status
  M-->>H: Sensor update
  O->>D: Mark delivery complete
  H->>H: Run automation
```

Events include parcel.created, parcel.status_changed, parcel.delivered, parcel.delayed, order.created, order.dispatched, refund.received and integration.failed. Use UUIDv7 event IDs, schema versions, timestamps and ownership metadata. At-least-once delivery; consumers deduplicate. Retry with backoff and dead-letter handling. Webhooks use signed payloads, replay protection and destination validation. SSE reconnects using event cursors with authorised replay.

MQTT Discovery sensors: active_parcels, arriving_today, out_for_delivery, delivered_today, delayed_parcels, next_delivery and online. Use dedicated broker credentials, topic ACLs, TLS where applicable, availability/LWT and minimal retained payloads; avoid exposing item names or tracking numbers by default. Remove retained discovery records when disconnected. HACS integration and dashboard card are later deliverables. Notifications: Home Assistant, ntfy, Gotify, email and webhooks. iOS APNs requires Apple infrastructure and a deliberate credential/distribution design; do not promise push without it. Rules engine supports triggers, conditions, quiet hours, deduplication and user consent.

## 10. User-facing features

Orders with multiple shipments, partial deliveries, replacements, cancellations and combined shipments; manual entry, courier detection with ambiguity review, share-sheet import, search, delivery history, returns and refunds, warranty reminders, local receipt storage, CSV/JSON export, optional spending summaries. Store attachments with size/type limits and access controls. Do not infer return or warranty deadlines without reliable evidence. Web dashboard, iOS widgets, Shortcuts and Live Activities are planned; native background execution is limited, so the server owns scheduling. Community frontends use the same API. Provide scoped SDKs and provider development docs; compile-in Go adapters initially rather than loading arbitrary executable plugins.

## 11. Deployment and operations

``` mermaid
flowchart TD
  C[Clients] --> VPN[HTTPS reverse proxy or private VPN]
  VPN --> GO[ParcelVault non-root container]
  GO --> VOL[(Persistent SQLite volume)]
  GO -. optional .-> PG[(PostgreSQL)]
  GO -. optional .-> AI[Ollama on trusted network]
  GO -. optional .-> MQTT[MQTT broker]
  MQTT --> HA[Home Assistant]
  GO --> EXT[Approved external services]
```

Docker Compose example for a future local build (not a published image):

``` yaml
services:
  parcelvault:
    build: .
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./data:/data
    environment:
      PARCELVAULT_DATABASE_DRIVER: sqlite
      PARCELVAULT_DATABASE_PATH: /data/parcelvault.db
```

Configuration names are proposed. Mount secrets separately; reverse proxy or VPN provides secure remote access. Support ARM64 and AMD64, graceful shutdown, migration locking, readiness and liveness, backup/restore and documented upgrades. SQLite backups must use a consistent backup mechanism, not a naive copy of an active database. Optional PostgreSQL and Valkey for larger deployments; no mandatory Redis. Local structured logs, queue depth, job latency, provider errors, quota usage, cache age and import health. No remote metrics export by default. Provide a security centre and external connection inventory. Retention and deletion controls include attachments, derived AI data and backup lifecycle.

## 12. Testing and release gates

Unit tests for normalisation, parsers and policy; fixture-based courier contract tests; integration tests for SQLite and PostgreSQL; fake clock for scheduler and token expiry; race detector and fuzzing for untrusted inputs; crash/restart and duplicate event tests; quota and 429 handling; backup restore drills; API compatibility checks; accessibility and iOS testing. CI builds multi-architecture images, runs go vet, govulncheck, tests and dependency scans. No public release until at least one real courier is authorised and security review is complete.

## 13. Roadmap

  ----------------------------------------------------------------------------------------------------------
  Phase                               Deliverable / acceptance gate
  ----------------------------------- ----------------------------------------------------------------------
  0                                   ADRs, threat model, OpenAPI draft, verify courier access

  1                                   Go scaffold, migrations, SQLite, secure bootstrap and JWT

  2                                   Parcel CRUD, UUIDv7, ownership tests and cached reads

  3                                   Persistent scheduler, fake provider, quota and restart tests

  4                                   First authorised courier integration and basic web UI

  5                                   Email imports, order/shipment model, review queue

  6                                   Local AI extraction and read-only assistant

  7                                   Outbox, MQTT, Home Assistant and notifications

  8                                   iOS client, SDKs, PostgreSQL parity and public alpha

  9                                   Returns/refunds, advanced rules, OIDC/passkeys, HACS, public release
  ----------------------------------------------------------------------------------------------------------

## 14. Architecture decisions and open questions

Accepted direction from discussion: Go, API-first, UUIDv7, JWT, persistent caching, self-hosting, local AI, Home Assistant, diagrams as documentation. Proposed implementation decisions: SQLite default, PostgreSQL optional, modular monolith, Ed25519, opaque refresh tokens, SSE, transactional outbox, React and SwiftUI, AGPL-3.0. Confirm licence, first courier and its quota/retention terms, initial deployment hardware, identity provider strategy, household sharing policy, APNs distribution, email OAuth approval and backup key management before implementation. The ParcelVault name is a working name; availability and trademarks have not been checked.

Maintain ADRs for each significant decision. Keep Mermaid diagrams in version control and update them alongside implementation changes. This document is a design draft, not a security audit, verified API integration or working application.
