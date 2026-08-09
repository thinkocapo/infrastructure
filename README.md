# infrastructure

<img src="sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

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
Dockerfile                    # builds the monitor into a container image (local build only)
docker-compose.yml            # example containers to monitor (postgres, redis, nginx) + optional monitor service
```

## Setup
```bash
go mod tidy
cp .env.example .env
# edit .env and add your SENTRY_DSN
```

## Run

### Run everything in Docker (sample containers + the monitor)

1. 
```bash
docker compose up -d --build
```

Now you have 3 containers for nginx, postgres, redix, and a 4th that's the monitor itself (Golang scripts in /collectors which use Sentry SDK's to gather metrics from the 3 containers)

### Run docker containers separately from the monitor

1. Starts Postgres, Redis, and Nginx only — skips `infra-monitor`, so it's safe to pair with `go run .` below without shipping duplicate metrics.
```bash
docker compose up -d postgres redis nginx
```

2. Start the monitor

Each source is a named collector (`host`, `docker`). Pick which ones run with the
`-collectors` flag (or the `COLLECTORS` env var). Order doesn't matter; empty = all.

```bash
go run . -collectors=host          # macOS host metrics only
go run . -collectors=docker        # Docker container metrics only
go run . -collectors=host,docker   # both (same as default)
go run .                           # all registered collectors
```

## Metrics

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

## Development - Adding a third source

The collector list lives in `collectors/registry.go`. To add one (Postgres, Redis,
Kubernetes, …):

1. Write a `CollectX(ctx context.Context)` function in a new file under `collectors/`
   (follow the shape of `host.go` / `docker.go` — read values, call `sentry.Metrics.Gauge`).
2. Add one line to `Registry`:
   ```go
   {Name: "postgres", Collect: CollectPostgres},
   ```

That's it — the `-collectors` flag, the run loop, and `-collectors=...` selection all pick it up automatically.
