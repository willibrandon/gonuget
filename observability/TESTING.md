# Observability Testing

## Unit Tests

Run unit tests (no external dependencies required):

```bash
go test ./observability -v -short
```

## Performance Benchmarks

Run logger performance benchmarks:

```bash
go test ./observability -bench=. -benchmem -run=^$
```

Key metrics:
- Standard logging: ~750-1000 ns/op
- Filtered debug logs: ~2 ns/op with zero allocations
- Null logger: ~0.2 ns/op with zero allocations

## Integration Tests

Integration tests require an OTLP collector (Jaeger) to be running.

### Option 1: Using Docker Compose (Recommended)

Start the full observability stack (Jaeger v2 + Prometheus):

```bash
cd observability
docker compose -f docker-compose.test.yml up -d
```

Wait for services to be ready (~5 seconds), then run all integration and E2E tests:

```bash
go test ./observability -v
```

Or run specific test suites:

```bash
# Integration tests only
go test ./observability -v -run TestSetupTracing_OTLP

# E2E tests only
go test ./observability -v -run TestE2E_
```

### Viewing Results

**Jaeger UI** (Traces):
- Open http://localhost:16686
- Available services:
  - `gonuget-integration-test` - from integration tests
  - `gonuget-e2e-test` - from E2E visualization test
  - `gonuget-full-stack-test` - from full stack test
- Click "Find Traces" to explore

**Prometheus UI** (Metrics):
- Open http://localhost:9090
- Query examples:
  - `gonuget_http_requests_total` - HTTP request counts
  - `gonuget_package_downloads_total` - Package download metrics
  - `gonuget_cache_hits_total` - Cache hit rates

Stop the stack:

```bash
docker compose -f docker-compose.test.yml down
```

### Option 2: Using Docker Run (Jaeger only)

For quick testing without Prometheus:

```bash
docker run -d -p 4317:4317 -p 16686:16686 --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/jaeger:latest

# Run tests
go test ./observability -v -run TestSetupTracing_OTLP

# View traces at http://localhost:16686

# Stop and remove container
docker stop jaeger && docker rm jaeger
```

Note: E2E Prometheus tests will be skipped without Prometheus running.

## Test Behavior

- **Unit tests** (`-short`) always run and test stdout/none exporters
- **Integration tests** automatically skip if collector is not available
- **E2E tests** verify full integration with Jaeger and Prometheus
- No test failures if services are missing - just skipped with helpful messages

## Coverage

Check test coverage:

```bash
go test ./observability -cover
```

View detailed coverage:

```bash
go test ./observability -coverprofile=coverage.out
go tool cover -html=coverage.out
```
