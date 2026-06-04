# infrastructure

Reads host metrics (CPU, memory, disk, network) from your MacBook and running Docker containers, then ships them to Sentry as application metrics.

The pattern for adding more sources (Kubernetes, Postgres, Redis, etc.) is just another collector file in `collectors/` — same loop, same Sentry emit calls. One Sentry project and DSN is enough; differentiate sources with tags.

## Two modes

Both point at the same Sentry project:

```bash
go run .                                              # direct SDK mode
otelcol-contrib --config otel_collector/config.yaml  # OTel Collector mode
```

**Direct SDK** — collection logic written in Go (`collectors/`), ships via Sentry SDK. More control, easier to hack on.

**OTel Collector** — pre-built receivers handle collection, ships via a custom Go exporter (`otel_collector/sentryexporter/`). The exporter was written from scratch since no official Sentry OTel metrics exporter exists — it receives `pmetric.Metrics` batches from the collector pipeline and translates each data point into a `sentry.Metrics.Gauge()` call. Vendor-neutral, closer to how production infra monitoring is set up. See `otel_collector/README.md` for setup.

## Structure

```
main.go                       # entry point — init Sentry, run collector loop
collectors/
  host.go                     # macOS host metrics via gopsutil
  docker.go                   # Docker container metrics via Docker Engine API
otel_collector/
  config.yaml                 # OTel Collector pipeline config
  README.md                   # setup instructions for OTel mode
docker-compose.yml            # example containers to monitor (postgres, redis, nginx)
python/                       # Python reference implementation
```

## Shared lineage with the Datadog agent

Several libraries used here are the same ones the Datadog agent uses internally — this project is essentially the same collection pipeline, pointed at Sentry instead of Datadog's backend.

| Library | Used here | Used by Datadog agent | Notes |
|---|---|---|---|
| `gopsutil/v3` | `collectors/host.go` | Yes | Reads CPU, memory, disk, network from the OS. Datadog uses this for their system-core check. |
| `docker/docker/client` | `collectors/docker.go` | Yes | Official Docker Go SDK. Both query the Docker Engine API over `/var/run/docker.sock`. |

The Datadog agent wraps these same libraries in ~450 "checks", schedules them every 15s, and forwards to Datadog's intake. This project does the same with two collectors and forwards to Sentry metrics.

## Setup

Install Go if needed:
```bash
brew install go
```

Then:
```bash
go mod tidy
cp python/.env.example .env
# edit .env and add your SENTRY_DSN
```

## Run

### Spin up example containers to monitor

```bash
docker compose up
```

Starts Postgres, Redis, and Nginx — all picked up automatically by `collectors/docker.go`.

### Start the monitor

```bash
go run .
```

Collects and emits metrics every 60 seconds (configurable via `INTERVAL_SECONDS` in `.env`).

### Selecting which sources to run

Each source is a named collector (`host`, `docker`). Pick which ones run with the
`-collectors` flag (or the `COLLECTORS` env var). Order doesn't matter; empty = all.

```bash
go run . -collectors=host          # macOS host metrics only
go run . -collectors=docker        # Docker container metrics only
go run . -collectors=host,docker   # both (same as default)
go run .                           # all registered collectors
```

```bash
COLLECTORS=host go run .           # same, via env (handy for launchd/Docker)
```

The monitor prints which collectors are active on startup, e.g.
`collectors: [host] — emitting every 60s`. Unknown names fail fast with the list
of valid collectors.

### Adding a third source

The collector list lives in `collectors/registry.go`. To add one (Postgres, Redis,
Kubernetes, …):

1. Write a `CollectX(ctx context.Context)` function in a new file under `collectors/`
   (follow the shape of `host.go` / `docker.go` — read values, call `sentry.Metrics.Gauge`).
2. Add one line to `Registry`:
   ```go
   {Name: "postgres", Collect: CollectPostgres},
   ```

That's it — the `-collectors` flag, the run loop, and `-collectors=...` selection all pick it up automatically.

## Scheduling

The program needs to run continuously (or be invoked repeatedly) to keep shipping metrics. Four options:

### 1. Built-in loop (default, recommended for demos)

The program already contains a `for` loop with a configurable sleep. Just run it once and leave it:

```bash
go run .
```

Set `INTERVAL_SECONDS=60` in `.env` to control the cadence. Kill with `Ctrl+C`.

### 2. cron

Invokes the binary as a new process on a schedule. Simple, but each run is independent — no persistent state.

```bash
crontab -e
# add:
* * * * * /usr/local/go/bin/go run /Users/you/thinkocapo/infrastructure
```

Note: if using this approach, remove the `for` loop so each cron invocation does a single collection pass.

### 3. launchd (macOS native, production-grade)

macOS's built-in job scheduler. More robust than cron — supports auto-restart on failure, logging, and boot-time launch. Build the binary first, then create a plist at `~/Library/LaunchAgents/com.thinkocapo.infrastructure.plist`:

```bash
go build -o infrastructure-monitor .
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.thinkocapo.infrastructure</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/you/thinkocapo/infrastructure/infrastructure-monitor</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>StandardOutPath</key>
    <string>/tmp/infrastructure.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/infrastructure.err</string>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

Load it with:
```bash
launchctl load ~/Library/LaunchAgents/com.thinkocapo.infrastructure.plist
```

### 4. Docker (self-contained, portable)

Run the Go binary in a container with the built-in loop. Docker Desktop on Mac can pass through `/var/run/docker.sock` so the container can still read stats from other running containers:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o infrastructure-monitor .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/infrastructure-monitor .
CMD ["./infrastructure-monitor"]
```

```bash
docker build -t infrastructure-monitor .
docker run -d \
  --env-file .env \
  -v /var/run/docker.sock:/var/run/docker.sock \
  infrastructure-monitor
```

The `-v /var/run/docker.sock` mount is what gives the container visibility into other containers' stats — without it, `collectors/docker.go` will get a connection error and skip Docker metrics gracefully.

---

## Metrics emitted

### Source 1: macOS host (`gopsutil`)

Reads directly from the macOS kernel via `gopsutil`.

| Metric | Tags |
|---|---|
| `host.cpu.percent` | `source`, `host` |
| `host.memory.used_mb` | `source`, `host` |
| `host.memory.percent` | `source`, `host` |
| `host.disk.used_gb` | `source`, `host` |
| `host.disk.percent` | `source`, `host` |
| `host.net.bytes_sent_mb` | `source`, `host` |
| `host.net.bytes_recv_mb` | `source`, `host` |

### Source 2: Docker containers (Docker Engine API)

Reads per-container stats from the Docker daemon. Each running container gets its own set of metrics, tagged by container name.

| Metric | Tags |
|---|---|
| `docker.cpu.percent` | `source`, `host`, `container` |
| `docker.memory.used_mb` | `source`, `host`, `container` |
| `docker.memory.percent` | `source`, `host`, `container` |

Docker metrics are skipped gracefully if Docker is not running.

## Querying by source in the Sentry UI

In the Sentry metrics explorer, use the `source` tag to filter or group by where metrics came from. Example tag schemes as more collectors are added:

```go
sentry.Metrics.Gauge("host.cpu.percent",   value, sentry.MetricTags({"source": "gopsutil",   "host": "macbook"}))
sentry.Metrics.Gauge("docker.cpu.percent", value, sentry.MetricTags({"source": "docker",     "container": "postgres"}))
sentry.Metrics.Gauge("k8s.pod.memory",     value, sentry.MetricTags({"source": "kubernetes", "namespace": "default"}))
```

In the UI: filter by `source = docker` to see only container metrics, or group by `container` to compare across containers. The `source` tag is the top-level discriminator; more specific tags (`host`, `container`, `namespace`) let you drill down within a source.

## OTel Collector mode — adding source tags via Processor

In the direct SDK mode (`go run .`), `source` tags are set explicitly in each collector. In the OTel Collector mode, `hostmetricsreceiver` and `dockerstatsreceiver` don't emit a `source` tag by default — you'd distinguish them only by metric name (e.g. `system.cpu.utilization` vs `container.cpu.percent`).

To stamp a `source` tag on each pipeline explicitly, split into two pipelines with an `attributes` processor on each:

```yaml
processors:
  attributes/host:
    actions:
      - key: source
        value: gopsutil
        action: insert
  attributes/docker:
    actions:
      - key: source
        value: docker
        action: insert

service:
  pipelines:
    metrics/host:
      receivers: [hostmetrics]
      processors: [attributes/host]
      exporters: [sentry]
    metrics/docker:
      receivers: [docker_stats]
      processors: [attributes/docker]
      exporters: [sentry]
```

This lets you filter by `source = gopsutil` or `source = docker` in the Sentry UI, matching the same tag structure used in direct SDK mode.
