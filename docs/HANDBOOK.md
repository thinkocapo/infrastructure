# Handbook

The questions that come up once someone's actually evaluating this — where this is headed, and what to check elsewhere in `/docs`. See [troubleshooting.md](troubleshooting.md) for what breaks and how to fix it. Two ways to run this exist: **Fleet-Wide** (one process, watches a whole host's containers) and **Per-Target** (one process per target, no Docker involved at all) — see the root [README](../README.md#run) if those terms are new.

## Benchmarking Overhead
[BenchmarkingOverhead.md](BenchmarkingOverhead.md) for performance overhead.

## Security Concerns
[SecurityConcerns.md](SecurityConcerns.md) for the security tradeoffs.

## Development

### Feature requests & roadmap

Known gaps surfaced so far, not yet built — [open a GitHub Issue](https://github.com/thinkocapo/infrastructure/issues/new) if any of these matter to you:

- A container/resource allow-list for Fleet-Wide, in both implementations — today it's all-or-nothing on whatever daemon it's pointed at (`dockerstatsreceiver`'s `excluded_images` gets partway there for OTel mode, but it's image-based exclusion, not a name-based allow-list).
- Retry/backoff on the OTel exporter — `exporterhelper.NewMetrics` isn't given `WithRetry`/`WithQueue` options today, so a failed export just errors out.
- A Kubernetes source — genuinely new work in either implementation, not a config change.
- Tagged releases for `otel_collector/sentryexporter`, so an external `ocb` manifest can pin a real version instead of a local `replaces` path.
- Histogram support in the OTel exporter (currently skipped).
- Real, measured performance-overhead numbers (see [BenchmarkingOverhead.md](BenchmarkingOverhead.md)).
- running Per-Target on a cloud VM

### Using this alongside Sentry SDKs

*Coming soon.* Most people trying this out are probably already running Sentry SDKs somewhere in their application stack for error monitoring and performance monitoring — worth writing up how this infrastructure monitor fits alongside that, plus guidance on distributed tracing setups so the two stories connect instead of living in separate silos.
