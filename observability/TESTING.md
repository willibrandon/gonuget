# Observability Testing

## Unit Tests

Run unit tests (no external dependencies required):

```bash
go test ./observability -v
```

## Integration Tests

Integration tests require an OTLP collector (Jaeger) to be running.

### Option 1: Using Docker Compose (Recommended)

Start the collector:

```bash
cd observability
docker-compose -f docker-compose.test.yml up -d
```

Wait for the collector to be ready (~5 seconds), then run tests:

```bash
go test ./observability -v -run TestSetupTracing_OTLP
```

View traces in Jaeger UI:
- Open http://localhost:16686
- Select service: `gonuget-integration-test`
- Click "Find Traces"

Stop the collector:

```bash
docker-compose -f docker-compose.test.yml down
```

### Option 2: Using Docker Run

```bash
docker run -d -p 4317:4317 -p 16686:16686 --name jaeger jaegertracing/all-in-one:latest

# Run tests
go test ./observability -v -run TestSetupTracing_OTLP

# View traces at http://localhost:16686

# Stop and remove container
docker stop jaeger && docker rm jaeger
```

## Test Behavior

- **Unit tests** always run and test stdout/none exporters
- **Integration tests** automatically skip if collector is not available
- No test failures if collector is missing - just skipped with helpful message

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
