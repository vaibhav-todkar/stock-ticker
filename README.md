# stock-ticker

A production-grade Go microservice that consumes live stock market data from the [Finnhub](https://finnhub.io) free API and prints/logs it in real-time.

---

## Architecture

```
cmd/
  app/
    main.go              ← Entry point, graceful shutdown
internal/
  config/                ← Viper-based config (YAML + env vars)
  logger/                ← Zap structured JSON logging
  client/                ← Finnhub HTTP client (retry, backoff, connection pooling)
  service/               ← Business logic, derived fields, unit tests
  model/                 ← Strongly-typed DTOs
  worker/                ← Goroutine-based polling + health check HTTP server
pkg/
  utils/                 ← Shared formatting utilities
config.yaml              ← Sample configuration
.env.example             ← Sample environment file
Dockerfile               ← Multi-stage Docker build
docker-compose.yml       ← Local development
Makefile                 ← build / run / test / lint / docker-*
```

---

## Prerequisites

- Go 1.24+
- A free [Finnhub API key](https://finnhub.io/register)
- Docker (optional)

---

## Getting a Finnhub API Key

1. Go to <https://finnhub.io/register> and create a free account.
2. Copy your API key from the dashboard.
3. Export it as an environment variable:

```bash
export FINNHUB_API_KEY=your_api_key_here
```

---

## Configuration

Configuration is loaded from `config.yaml` in the working directory, then overridden by environment variables.

| Key                    | Env var              | Default                        | Description                       |
|------------------------|----------------------|--------------------------------|-----------------------------------|
| `api.base_url`         | `API_BASE_URL`       | `https://finnhub.io/api/v1`    | Finnhub REST base URL             |
| `api.api_key`          | `FINNHUB_API_KEY`    | _(required)_                   | Finnhub API key                   |
| `api.symbols`          | `API_SYMBOLS`        | AAPL, GOOGL, MSFT, TSLA, AMZN | Stock symbols to track            |
| `api.poll_interval`    | `API_POLL_INTERVAL`  | `10s`                          | How often to poll the API         |
| `api.timeout`          | `API_TIMEOUT`        | `10s`                          | HTTP request timeout              |
| `api.max_retries`      | `API_MAX_RETRIES`    | `3`                            | Max retries with exponential back-off |
| `server.port`          | `SERVER_PORT`        | `8080`                         | Health check HTTP port            |
| `logging.level`        | `LOGGING_LEVEL`      | `info`                         | Log level (debug/info/warn/error) |
| `logging.format`       | `LOGGING_FORMAT`     | `json`                         | Log format (json/console)         |

Edit `config.yaml` for file-based overrides, or copy `.env.example` to `.env` and load it.

---

## Build & Run

### Locally

```bash
# Install dependencies
go mod download

# Build
make build

# Run (FINNHUB_API_KEY must be set)
export FINNHUB_API_KEY=your_api_key_here
make run
# or directly:
./bin/stock-ticker
```

### Tests

```bash
make test
```

### Lint

```bash
# Requires golangci-lint: https://golangci-lint.run/usage/install/
make lint
```

---

## Docker

```bash
# Build the image
make docker-build

# Run the container
export FINNHUB_API_KEY=your_api_key_here
make docker-run
```

### docker-compose

```bash
FINNHUB_API_KEY=your_api_key_here docker-compose up
```

---

## Example Output

```
[2026-04-09T10:30:00Z] [INFO] Symbol: AAPL   | Price: $182.50     | Change: +2.35 (+1.30%) | High: $183.10 | Low: $180.20
[2026-04-09T10:30:00Z] [INFO] Symbol: GOOGL  | Price: $141.80     | Change: -0.45 (-0.32%) | High: $142.50 | Low: $140.90
[2026-04-09T10:30:00Z] [INFO] Symbol: MSFT   | Price: $310.00     | Change: +1.50 (+0.49%) | High: $311.20 | Low: $308.00
```

JSON log lines (from Zap) are also written to stdout:

```json
{"level":"info","ts":1744192200.123,"msg":"fetched quote","symbol":"AAPL","price":182.5,"change":2.35,"change_percent":1.30}
```

---

## Health Check

The service exposes a `/health` endpoint:

```bash
curl http://localhost:8080/health
# {"status":"ok","uptime":"5m0s","symbols_tracked":5}
```

---

## Security

- The API key is **never** hardcoded. Always pass it via the `FINNHUB_API_KEY` environment variable.
- `.env` files are in `.gitignore`.
