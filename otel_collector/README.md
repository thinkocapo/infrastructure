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

### What OTel data types map to what

| OTel metric type | Mapped to Sentry |
|---|---|
| `Gauge` | `sentry.Metrics.Gauge()` |
| `Sum` (counter) | `sentry.Metrics.Gauge()` (Sentry has no counter type) |
| `Histogram` | skipped (not supported yet) |

## Files

```
otel_collector/
  main.go                  # custom collector binary — wires receivers + sentryexporter
  config.yaml              # pipeline config
  sentryexporter/
    exporter.go            # ConsumeMetrics() — translates OTel metrics to Sentry
    factory.go             # registers the exporter with the collector
    config.go              # DSN config
```

## Build and run

```bash
cd otel_collector

# build the custom collector binary
go build -o otelcol-sentry .

# run it
export SENTRY_DSN=https://a0fb37cd705816a19852120edcd719c9@o262702.ingest.us.sentry.io/4511492247584768
./otelcol-sentry --config config.yaml
```

## Comparing the two modes

| | Go SDK (`go run .`) | OTel Collector (`./otelcol-sentry`) |
|---|---|---|
| Collection logic | written in Go (`collectors/`) | pre-built receivers in config |
| Flexibility | full control over what/how | limited to receiver capabilities |
| Vendor lock-in | Sentry SDK only | swap exporter to ship anywhere |
| What to run | `go run .` from repo root | `./otelcol-sentry --config config.yaml` |
| Good for | learning, custom metrics | production, vendor-neutral pipelines |
