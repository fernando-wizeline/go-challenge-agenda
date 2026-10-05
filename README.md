# Clinic Scheduling Challenge

A monorepo with two services that power a medical scheduling system.

## Architecture

```
services/agenda   — internal gRPC service, owns all scheduling logic and data
services/api      — public HTTP REST API (Gin), calls agenda via gRPC
```

Both services share a single Go module. Proto definitions live in `proto/` and generated code in `gen/`.

## Quick start

```bash
# Generate proto code (requires buf)
make proto

# Run with Docker Compose
make docker-up

# API available at http://localhost:8080
# Agenda gRPC at localhost:50051
# Agenda Prometheus metrics at http://localhost:9090/metrics
```

## Configuration (agenda service)

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENDA_GRPC_ADDR` | `:50051` | gRPC listen address |
| `AGENDA_METRICS_ADDR` | `:9090` | Prometheus `/metrics` listen address |
| `DB_DRIVER` / `DB_SOURCE` | `sqlite3` / `agenda.db` | Database driver and DSN |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |

## Metrics

The agenda service exposes Prometheus metrics at `/metrics` on `AGENDA_METRICS_ADDR`, along with Go runtime and process metrics.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `agenda_reservation_attempts_total` | counter | `result` | Reservation attempts by outcome: `created`, `slot_conflict`, `slot_blocked`, `patient_not_found`, `error` |
| `agenda_availability_duration_seconds` | histogram | `result` (`ok`/`error`) | Time spent computing doctor availability |

How to read them:

- `slot_conflict` as a share of attempts shows how often patients are offered times that are already taken.
- `slot_blocked` should stay near zero, because availability already hides blocked time. If it doesn't, availability and booking disagree about blocks.
- `error` is the one to alert on. The other non-`created` results are normal business rejections.

```bash
curl -s localhost:9090/metrics | grep '^agenda_'
```

## API routes

| Method | Path | Description |
|--------|------|-------------|
| GET | /v1/doctors | List all doctors |
| GET | /v1/doctors/:id | Get doctor with working hours |
| GET | /v1/doctors/:id/availability?date=YYYY-MM-DD&type=first_visit | Get available slots |
| POST | /v1/reservations | Create a reservation |
| GET | /v1/reservations/:id | Get reservation |
| GET | /v1/reservations?doctor_id=&from=&to= | List reservations |
| DELETE | /v1/reservations/:id | Cancel reservation |
| GET | /v1/users | List all users |
| POST | /v1/users | Create a user |
| GET | /v1/users/:id | Get user |
| PATCH | /v1/users/:id | Update user |
| DELETE | /v1/users/:id | Delete user |
| GET | /v1/users/:id/reservations | List reservations for a user |

## Reservation types

- `first_visit` — 60 minute block
- `follow_up` — 30 minute block

## Tasks

Work through as many issues as you can. They are loosely ordered from more concrete to more open-ended.

- [ ] **Issue #1** — `GET /v1/doctors/:id/availability` returns hardcoded stub data. Wire it to the real usecase.
- [ ] **Issue #2** — There is a bug in reservation conflict detection that allows double-booking in certain cases. Find and fix it.
- [ ] **Issue #3** — `ListReservations` in the SQLite repository is not implemented. Implement it.
- [ ] **Issue #4** — gRPC errors are not properly mapped to HTTP status codes. Fix the mapping in `pkg/errcodes`.
- [ ] **Issue #5** — There are failing tests in the codebase. Make them pass.
- [ ] **Issue #6** — `GET /v1/users/:id/reservations` returns `501 Not Implemented`. Implement it end-to-end across all layers.
- [ ] **Issue #7** — The `api` service usecases are coupled to a concrete type where an interface should be used. Fix it.
- [ ] **Issue #8** — Blocked slots are stored and retrieved, but recurrences are never expanded. Implement recurrence expansion (daily / weekly / monthly until a given date).
- [ ] **Issue #9** — The system only supports SQLite. Add Postgres support, switchable via `DB_DRIVER=postgres` environment variable, without modifying business logic.
- [ ] **Issue #10** — Some HTTP handlers bypass the usecase layer. Fix the layering.
- [ ] **Issue #11** — Blocked slots are not taken into account when computing availability. Fix this.
- [ ] **Issue #12** — Extend the system to support additional service types (labs, therapy) with different slot durations. Design the domain model, update the proto, and expose them through the existing services.
- [ ] **Issue #13** — Add structured logging and at least one meaningful metric without coupling observability concerns to the domain layer.
- [ ] **Issue #14 (optional)** — Consider whether the system would benefit from a dedicated third service. If so, propose the service boundary, define its proto contract, and implement it.
- [ ] **Issue #15** — Write a short ADR (Architecture Decision Record) for one technical decision you made or changed during this challenge.

## Pre-seeded doctors

| ID | Name | Specialty | Working days |
|----|------|-----------|-------------|
| doc-001 | Dr. Ana García | General Practice | Mon–Fri (09–17, Fri until 13) |
| doc-002 | Dr. Luis Mendoza | Cardiology | Mon, Wed, Fri (08–16, Fri until 12) |
| doc-003 | Dr. Sara Patel | Pediatrics | Tue, Thu (10–18) |

## Logging ADR

### Context

The availability and booking flows had no structured logging. We wanted `log/slog` logs without coupling observability concerns to the domain layer, and without scattering timing and status code logging across every gRPC method.

### Decision

Logging lives outside the domain and the usecases, in `services/agenda/internal/observability`, and is attached from the outside in `cmd/main.go`:

1. **Decorators for business outcomes.** `ObservedAvailability` and `ObservedReservation` wrap the usecases, log the result, record metrics, and delegate. Usecases and `domain/` do not import `slog` or Prometheus.
2. **A gRPC unary interceptor for transport outcomes.** `UnaryLoggingInterceptor` logs method, status code and duration for every RPC, so new RPCs are covered automatically.
3. **Interfaces on the gRPC server.** `Server` now depends on small consumer-side interfaces, `AvailabilityService` and `ReservationService`, instead of the concrete `*usecase.AvailabilityUsecase` and `*usecase.ReservationUsecase`. A decorator is a different type from the usecase, so without the interfaces it could not be plugged in. The usecases satisfy the interfaces implicitly and did not change.
4. **Distinguishable rejections.** A booking rejected by a blocked slot returns `domain.ErrSlotBlocked`, which wraps `ErrSlotNotAvailable`. `errors.Is` checks and the HTTP 409 mapping are unchanged, but the log and the metric can say why the booking was rejected.
5. **Configuration.** `LOG_LEVEL` (default `info`) and `LOG_FORMAT` (`json` or `text`, default `json`).
6. **Metrics.** Two Prometheus metrics are recorded in the same decorators: `agenda_reservation_attempts_total{result}` and `agenda_availability_duration_seconds{result}`, served on a separate port (`AGENDA_METRICS_ADDR`, default `:9090`) because the agenda service only speaks gRPC.

Logging rules: each event is logged once, at one boundary; only ids are logged, never patient name, phone or email; expected business rejections (slot conflict, blocked slot, unknown patient) are `Warn`, and everything unexpected is `Error`.

### Alternatives considered

- **Inject a `*slog.Logger` into each usecase and log inline.** This is the simplest option, but it mixes logging into business logic and forces every usecase test to supply a logger.
- **An interceptor only.** It cannot see business reasons such as a blocked slot or an unknown patient, only the final gRPC code.

### Consequences

- Usecases and domain stay free of logging, and logging can be switched off by wiring in the undecorated usecases.
- One conflicting booking produces two log lines: a `Warn` from the decorator saying why, and an `Info` from the interceptor with the status code and duration.
- Each usecase method that should be logged needs a decorator method. Today only availability and booking (`Create`, `Cancel`) are covered; blocked-slot, doctor and patient flows are not.
- The API service is not covered. Request ids across the API-to-agenda gRPC call are a possible future step.
- Metrics reuse the same decorators and the same error classification (see Metrics above), so the decorators are now named `Observed*` rather than `Logging*`. A nil `*Metrics` is valid and records nothing.
