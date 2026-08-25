# Siraaj Analytics

Privacy-first, self-hosted web analytics platform built in Go + DuckDB. Lightweight alternative to Google Analytics.

## Tech Stack

- **Backend**: Go 1.24+, DuckDB 2.5+ (embedded)
- **Frontend**: SvelteKit 5+, Tailwind CSS 4+, D3.js v7+
- **SDK**: Vanilla JS/TS (<16KB gzipped), Rollup bundling
- **Dev**: Air (hot reload), MockGen, Testify

## Project Structure

```
internal/
  domain/          # Domain models (Event, Stats, etc.)
  service/         # Business logic, EventService interface
  repository/      # DuckDB data access, EventRepository interface
  handler/         # HTTP handlers
  middleware/       # CORS, BasicAuth, Logging
  migrations/      # DB schema & indexes
  botdetector/     # Bot detection
  channeldetector/ # Traffic channel classification
  mocks/           # MockGen generated mocks
  tests/           # Integration tests & benchmarks
dashboard/         # SvelteKit frontend
sdk/               # JavaScript analytics SDK
ui/                # Compiled assets embedded in binary
geolocation/       # GeoIP lookup service
docs/              # VitePress documentation
data/              # Runtime DB and GeoIP storage
```

## Common Commands

```bash
# Development
air                          # Hot reload dev server

# Build
go build -o siraaj .

# Testing
make test                    # All tests
make test-unit               # Unit tests only
make test-integration        # Integration tests only
make test-coverage           # HTML coverage report
make test-race               # Race detector
make bench                   # Benchmarks
make test-ci                 # CI mode (coverage + race)

# Mocks
make mocks                   # Regenerate mocks

# Docker
docker build -t siraaj .
docker run -d -p 8080:8080 -v $(pwd)/data:/data siraaj
```

## Environment Variables

```bash
PORT=8080
DB_PATH=data/analytics.db
DUCKDB_MEMORY_LIMIT=4GB
DUCKDB_THREADS=4
DASHBOARD_USERNAME=admin      # Set both to enable auth
DASHBOARD_PASSWORD=password
CORS=https://example.com      # Default: *
GEODB_PATH=data/geodb/dbip.mmdb
```

## Architecture

Clean architecture with strict layer separation:
1. **Domain** → pure models, no dependencies
2. **Repository** → DuckDB operations, implements interface
3. **Service** → business logic, depends on repository interface
4. **Handler** → HTTP, depends on service interface

Key patterns:
- Dependency injection via interfaces (enables mock testing)
- Batch inserts (5000 records) for DuckDB performance
- Go `embed` package for UI assets embedded in binary
- Multi-tenancy via `project_id` on all events

## Database Notes

- DuckDB is embedded (no external DB process)
- Uses sequences for ID generation
- Strategic covering indexes on date partitions (hourly/daily/monthly)
- Schema migrations in `internal/migrations/`
