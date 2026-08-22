# infrastructure

Collects host metrics (CPU, memory, disk, network) and Docker container stats, then ships them to Sentry as application metrics — using the Sentry SDK directly, with no separate monitoring agent to deploy. See `collectors/` for how it's done via `gopsutil` and the Docker Engine API. There's also a OTel Collector-based [implementation](otel_collector/README.md) available. No Kubernetes support yet in either implementation — that's a real collector/receiver to build, not a config tweak. I'm looking for initial feedback on the current state of this project first.

[Handbook in docs/ covers Overhead, Troubleshooting, Security, Roadmap, Sentry Overview](docs/HANDBOOK.md).

## Sentry Metrics
[Sentry Metrics](./docs/sentry-metrics.md)  

<img src="./sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

## Setup
```bash
# you'll need a SENTRY_DSN in your .env
# collection interval defaults to 60s — tune it via INTERVAL_SECONDS in .env
go mod tidy
cp .env.example .env
```

## Run

### Run Demo App

`docker-compose.yml` spins up 4 containers: Postgres, Redis, and Nginx as sample workloads to monitor, plus `infra-monitor` — the monitor itself, containerized, watching the other three via the Docker Engine API. This is the Fleet-Wide approach — it's 1 monitor watching the whole daemon.

```bash
docker compose up -d --build
```

`infra-monitor` can also be run separately from the 3 sample containers — useful while rebuilding or iterating on it independently:
```bash
docker compose up -d postgres redis nginx
docker compose up -d --build infra-monitor
```

### Operations: Run Fleet-Wide

One monitor instance per host, talking to that host's Docker socket and reporting on every container the daemon can see — exactly what the Demo App above does, via `docker-compose.yml`. To run it standalone, outside of Compose:

```bash
docker build -t infrastructure-monitor .
docker run -d --restart unless-stopped \
  --env-file .env \
  -e COLLECTORS=docker \
  -e CONTAINERS=name1,name2 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  infrastructure-monitor
```

`CONTAINERS` is optional — a comma-separated allow-list of container names; unset means every container on the daemon, as before. Drop it if you want everything.

`docker-compose.yml` uses a safer pattern than the raw socket mount above: a read-only [`docker-proxy`](docker-compose.yml) service fronts the real socket, and `infra-monitor` talks to that instead (`DOCKER_HOST=tcp://docker-proxy:2375`) — no direct socket mount on the monitor at all. See [Security Concerns](docs/SecurityConcerns.md) for why that matters.

### Developers: Run Per-Target

Instead of one monitor watching a whole host's containers, run one monitor instance *inside* each container (or VM) you actually want visibility into, with only the host collector enabled — `gopsutil` just reads `/proc`-level stats for wherever it's running, so it never touches the Docker socket at all. That makes it naturally scoped to exactly one target, with no socket-access question to raise.

Steps:
1. Build the binary:
   ```bash
   go build -o infrastructure-monitor .
   ```
2. Copy `infrastructure-monitor` into whatever the target container image already builds (a `COPY` line in your own Dockerfile), or drop it directly onto a VM.
3. Run it alongside the existing process, no Docker dependency at all:
   ```bash
   SENTRY_DSN=... COLLECTORS=host ./infrastructure-monitor
   ```
4. Repeat per container or host that needs visibility. Each instance tags its metrics with its own real hostname, so multiple instances stay distinguishable in the Sentry UI without extra config.

[Security Concern](docs/SecurityConcerns.md) for the tradeoff between Fleet-Wide and Per-Target.

## Test Pathway to Production

Guidance on what to test this on, and in what order — so it feels like running a proof-of-concept, not a leap of faith.

### Developers

1. Your own local dev / self-hosted.
2. A staging environment that mirrors prod topology.
3. One well-bounded, non-customer-facing background service in real production, after reviewing [Security Concerns](docs/SecurityConcerns.md).

### Operations

Less a strict 1-2-3 order, more a set of things to work through:

1. Review [Security Concerns](docs/SecurityConcerns.md) and check whether a formal security review is needed.
2. Benchmark overhead — see [docs/BenchmarkingOverhead.md](docs/BenchmarkingOverhead.md).
3. Test on-call/alerting integration.
4. Kubernetes is not yet supported.