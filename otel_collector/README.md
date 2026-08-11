# OTel Collector mode

An alternative to the Go SDK approach in `main.go`. Instead of writing collection logic in code, the OTel Collector handles it via config — with pre-built receivers for host metrics and Docker stats.

Both modes collect the same data and ship to the same Sentry project. This one is closer to how real production infra monitoring is set up.

## How it works

```
hostmetricsreceiver  }
                     }→  OTel Collector  →  sentryexporter (custom)  →  Sentry metrics API
dockerstatsreceiver  }
```

The key piece is `sentryexporter/` — a custom OTel exporter that:
1. Receives OTel `pmetric.Metrics` batches from the collector pipeline
2. Walks the metrics tree (ResourceMetrics → ScopeMetrics → Metric → DataPoints)
3. Translates each data point into a `sentry.Metrics.Gauge()` call
4. Tags are pulled from both resource-level attributes (e.g. `host.name`) and datapoint-level attributes (e.g. `container.name`)

`sentryexporter/` is its own Go module (`otel_collector/sentryexporter/go.mod`) — a real, independent OTel Collector exporter component, not something that only works inside this repo's custom `main.go` binary. That's what makes the "add it to your own collector" path below possible.

### What OTel data types map to what

| OTel metric type | Mapped to Sentry |
|---|---|
| `Gauge` | `sentry.Metrics.Gauge()` |
| `Sum` (counter) | `sentry.Metrics.Gauge()` (Sentry has no counter type) |
| `Histogram` | skipped (not supported yet) |

## Files

```
otel_collector/
  main.go                  # demo binary — wires receivers + sentryexporter
  config.yaml              # demo pipeline config
  builder-config.yaml      # ocb manifest — assembles sentryexporter into its own distribution
  sentryexporter/
    go.mod, go.sum         # independent module — no dependency on the rest of this repo
    exporter.go            # ConsumeMetrics() — translates OTel metrics to Sentry
    factory.go             # registers the exporter with the collector
    config.go              # DSN config
```

## Run Demo

The demo binary hand-wires `hostmetricsreceiver`, `dockerstatsreceiver`, and `sentryexporter` together in Go code — the fastest way to try this out locally, but not how you'd add `sentryexporter` to a collector you already run (see below for that).

It reads `SENTRY_DSN` from the shell environment — it doesn't use `godotenv` like the root `main.go` does. Use `source` to load the root `.env` file (already gitignored) before running:

```bash
cd otel_collector

# build the demo binary
go build -o otelcol-sentry .

# load DSN from root .env and run
source ../.env && ./otelcol-sentry --config config.yaml
```

No DSN in shell history, nothing committed to source.

## Add sentryexporter to your own OTel Collector

`builder-config.yaml` is a manifest for the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) (`ocb`) — the standard tool most real OTel Collector distributions (including the official Contrib distro) use to assemble a binary from a declarative list of components. This proves `sentryexporter` is a real, pluggable component: the manifest below builds a distribution from official upstream receivers plus `sentryexporter`, side by side, the same way you'd add it to a collector you already build and run.

```bash
# install ocb once, matching this repo's pinned collector version
go install go.opentelemetry.io/collector/cmd/builder@v0.153.0

# this repo has a root go.work for local multi-module development —
# GOWORK=off makes ocb resolve modules on its own instead
cd otel_collector
GOWORK=off "$(go env GOPATH)/bin/builder" --config builder-config.yaml
```

This produces `otel_collector/dist/otelcol-sentry-custom` (gitignored — it's a build artifact). Verify `sentry` registered as a component, alongside the official ones:
```bash
./dist/otelcol-sentry-custom components
```

`sentryexporter` isn't tagged/published yet, so the manifest's `replaces` entry points at the local checkout instead of a real module version:
```yaml
replaces:
  - github.com/thinkocapo/infrastructure/otel_collector/sentryexporter => ../sentryexporter
```
Once a release is tagged, drop `replaces` and pin the exporter's `gomod` line to that version instead — an external consumer's own manifest would look identical otherwise.

## Comparing the two modes

| | Go SDK (`go run .`) | OTel Collector (`./otelcol-sentry`) |
|---|---|---|
| Collection logic | written in Go (`collectors/`) | pre-built receivers in config |
| Flexibility | full control over what/how | limited to receiver capabilities |
| Vendor lock-in | Sentry SDK only | swap exporter to ship anywhere |
| What to run | `go run .` from repo root | `./otelcol-sentry --config config.yaml` |
| Good for | learning, custom metrics | production, vendor-neutral pipelines |

## OTel Collector mode — adding source tags via Processor

In the direct SDK mode (`go run .` from the repo root), `source` tags are set explicitly in each collector. In the OTel Collector mode, `hostmetricsreceiver` and `dockerstatsreceiver` don't emit a `source` tag by default — you'd distinguish them only by metric name (e.g. `system.cpu.utilization` vs `container.cpu.percent`).

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
