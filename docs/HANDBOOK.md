# Handbook

The questions that come up once someone's actually evaluating this — overhead, what breaks and how to fix it, security, and where this is headed. Two ways to run this exist: **Fleet-Wide** (one process, watches a whole host's containers) and **Per-Target** (one process per target, no Docker involved at all) — see the root [README](../README.md#run) if those terms are new.

## Overhead

Not formally benchmarked yet with real numbers — that's on the roadmap (see Development below), and the plan is to measure it on an actual cloud VM rather than guess. Qualitatively, here's the shape:

- **Per-Target / Direct SDK:** a small, statically-built Go binary doing a handful of `gopsutil` reads (CPU, memory, disk, network) once per interval. No runtime to speak of — no JVM, no interpreter — so the floor is very low, in the same ballpark as any lightweight Go CLI tool idling between ticks.
- **Fleet-Wide / Direct SDK:** the same, plus one Docker Engine API call per container per interval. Scales linearly with container count on that host.
- **OTel Collector mode:** a heavier baseline than the direct SDK binary, since it's a full OTel Collector process (pipeline framework, receivers, batching) rather than a minimal purpose-built binary — how much heavier isn't measured yet.

Once real numbers exist (CPU/memory of the `infra-monitor` container or process itself, ideally from a real VM rather than a laptop), they belong here instead of this qualitative description.

## Troubleshooting

Real failure modes hit and fixed during development — not hypothetical:

- **`[docker] unavailable: ...` / Docker socket unreachable.** `CollectDocker` (or `dockerstatsreceiver` in OTel mode) can't reach the daemon. Self-monitoring reports `monitor.collector.up = 0` and captures a Sentry issue tagged `collector: docker` when this happens — check that Docker is running, and that `/var/run/docker.sock` is actually mounted if you're running `infra-monitor` itself as a container.
- **Missing `SENTRY_DSN`.** Direct SDK mode fails fast: `log.Fatal("SENTRY_DSN is required")`. OTel Collector mode fails differently — it warns `Configuration references unset environment variable`, then hard-errors with `invalid configuration: exporters::sentry: dsn is required`. Same root cause, different failure shape depending on which implementation you're running.
- **`source ../.env` doesn't actually set the DSN when running the OTel demo binary.** `.env` doesn't use `export`, so a plain `source` only sets a shell variable — it isn't inherited by the child process. Use `set -a; source ../.env; set +a` instead (or `export $(cat ../.env | xargs)`), so the variables actually propagate.
- **Building an OTel Collector distribution via `ocb` fails with `main module ... does not contain package ...`.** This repo has a root `go.work` for local multi-module development, which captures ocb's generated output directory as if it were part of the root module. Run `builder` with `GOWORK=off` to bypass the workspace for that build.
- **`host` tag shows `unknown`.** `HostTag()` falls back to `"unknown"` when `os.Hostname()` itself fails — rare, but can happen in a locked-down container. If you see it, check whether the environment restricts the hostname syscall.

## Security

**Docker socket access.** `-collectors=docker` (Fleet-Wide) requires mounting `/var/run/docker.sock` into the monitor's container, which is root-equivalent access to that host — a process with access to that socket can create, exec into, or mount the host filesystem into any container. Worth raising explicitly with whoever owns security for a given environment, not something to hand over quietly.

**Per-Target vs. Fleet-Wide tradeoff.** Choosing Per-Target means running one process per target instead of one process watching the whole fleet — more instances to deploy and update, in exchange for no socket exposure and no all-or-nothing visibility. `-collectors=docker` stays fully available in this same binary for local testing or any environment where the socket tradeoff is acceptable — [open a feature request](https://github.com/thinkocapo/infrastructure/issues/new) if there's something you'd like to see to make this more usable for you.

## Development

### Feature requests & roadmap

Known gaps surfaced so far, not yet built — [open a GitHub Issue](https://github.com/thinkocapo/infrastructure/issues/new) if any of these matter to you:

- A container/resource allow-list for Fleet-Wide, in both implementations — today it's all-or-nothing on whatever daemon it's pointed at (`dockerstatsreceiver`'s `excluded_images` gets partway there for OTel mode, but it's image-based exclusion, not a name-based allow-list).
- Retry/backoff on the OTel exporter — `exporterhelper.NewMetrics` isn't given `WithRetry`/`WithQueue` options today, so a failed export just errors out.
- A Kubernetes source — genuinely new work in either implementation, not a config change.
- Tagged releases for `otel_collector/sentryexporter`, so an external `ocb` manifest can pin a real version instead of a local `replaces` path.
- Histogram support in the OTel exporter (currently skipped).
- Real, measured performance-overhead numbers (see Overhead above).

### Using this alongside Sentry SDKs

*Coming soon.* Most people trying this out are probably already running Sentry SDKs somewhere in their application stack for error monitoring and performance monitoring — worth writing up how this infrastructure monitor fits alongside that, plus guidance on distributed tracing setups so the two stories connect instead of living in separate silos.
