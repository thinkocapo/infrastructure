# OTel Collector mode

An alternative to the Go SDK approach in `main.go`. Instead of writing collection logic in code, the OTel Collector handles it via config — with pre-built receivers for host metrics and Docker stats.

Both modes collect the same data and ship to the same Sentry project. This one is closer to how real production infra monitoring is set up.

## How it works

```
hostmetricsreceiver  }
                     }→  OTel Collector  →  Sentry OTel exporter  →  Sentry.io
dockerstatsreceiver  }
```

No Go code involved — just the collector binary + `config.yaml`.

## Setup

### 1. Install the OTel Collector contrib distro

The contrib distro includes the `docker_stats` receiver and Sentry exporter. Install via Homebrew:

```bash
brew install opentelemetry-collector-contrib
```

Or download directly from:
https://github.com/open-telemetry/opentelemetry-collector-contrib/releases

### 2. Set your DSN

The config reads `SENTRY_DSN` from the environment:

```bash
export SENTRY_DSN=https://a0fb37cd705816a19852120edcd719c9@o262702.ingest.us.sentry.io/4511492247584768
```

Or add it to your shell profile.

### 3. Run the collector

```bash
otelcol-contrib --config ./otel_collector/config.yaml
```

## Comparing the two modes

| | Go SDK (`go run .`) | OTel Collector |
|---|---|---|
| Collection logic | written in Go (`collectors/`) | pre-built receivers in config |
| Flexibility | full control over what/how | limited to receiver capabilities |
| Vendor lock-in | Sentry SDK only | swap exporter to ship anywhere |
| What to run | `go run .` | `otelcol-contrib --config ...` |
| Good for | learning, custom metrics | production, vendor-neutral pipelines |
