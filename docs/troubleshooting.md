# Troubleshooting

Real failure modes hit and fixed during development — not hypothetical:

- **`[docker] unavailable: ...` / Docker socket unreachable.** `CollectDocker` (or `dockerstatsreceiver` in OTel mode) can't reach the daemon. Self-monitoring reports `monitor.collector.up = 0` and captures a Sentry issue tagged `collector: docker` when this happens — check that Docker is running, and that `/var/run/docker.sock` is actually mounted if you're running `infra-monitor` itself as a container.
- **Missing `SENTRY_DSN`.** Direct SDK mode fails fast: `log.Fatal("SENTRY_DSN is required")`. OTel Collector mode fails differently — it warns `Configuration references unset environment variable`, then hard-errors with `invalid configuration: exporters::sentry: dsn is required`. Same root cause, different failure shape depending on which implementation you're running.
- **`source ../.env` doesn't actually set the DSN when running the OTel demo binary.** `.env` doesn't use `export`, so a plain `source` only sets a shell variable — it isn't inherited by the child process. Use `set -a; source ../.env; set +a` instead (or `export $(cat ../.env | xargs)`), so the variables actually propagate.
- **Building an OTel Collector distribution via `ocb` fails with `main module ... does not contain package ...`.** This repo has a root `go.work` for local multi-module development, which captures ocb's generated output directory as if it were part of the root module. Run `builder` with `GOWORK=off` to bypass the workspace for that build.
- **`host` tag shows `unknown`.** `HostTag()` falls back to `"unknown"` when `os.Hostname()` itself fails — rare, but can happen in a locked-down container. If you see it, check whether the environment restricts the hostname syscall.
