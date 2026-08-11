# infrastructure

<img src="sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

Collects host metrics (CPU, memory, disk, network) and Docker container stats, then ships them to Sentry as application metrics — using the Sentry SDK directly, with no separate monitoring agent to deploy. See `collectors/` for how it's done via `gopsutil` and the Docker Engine API. [Sentry Metrics](#sentry-metrics)

## Setup
```bash
# you'll need a SENTRY_DSN to add to .env below
# collection interval defaults to 60s — tune it via INTERVAL_SECONDS in .env
go mod tidy
cp .env.example .env
```

## Run

### Run Demo App

`docker-compose.yml` spins up 4 containers: Postgres, Redis, and Nginx as sample workloads to monitor, plus `infra-monitor` — the monitor itself, containerized, watching the other three via the Docker Engine API.

```bash
docker compose up -d --build
```

`infra-monitor` can also be run separately from the 3 sample containers — useful while rebuilding or iterating on it independently:
```bash
docker compose up -d postgres redis nginx
```

Or rebuild and restart just `infra-monitor` on its own (starts its 3 dependencies too if they're not already running, but only rebuilds the monitor's image):
```bash
docker compose up -d --build infra-monitor
```

### Run Fleet-Wide

One monitor instance per host, talking to that host's Docker socket and reporting on every container the daemon can see — exactly what the Demo App above does, via `docker-compose.yml`. To run it standalone, outside of Compose:

```bash
docker build -t infrastructure-monitor .
docker run -d --restart unless-stopped \
  --env-file .env \
  -e COLLECTORS=docker \
  -v /var/run/docker.sock:/var/run/docker.sock \
  infrastructure-monitor
```

Two things to know before pointing this at a real environment:
- It has no way to scope to specific containers — it reports on the whole daemon, all or nothing. [Open a GitHub Issue](https://github.com/thinkocapo/infrastructure/issues/new) if you have a feature request.
- It requires mounting `/var/run/docker.sock` into the container, which is root-equivalent access to that host — worth raising explicitly with whoever owns security for that environment, not something to hand over quietly.

### Run Per-Target

Instead of one monitor watching a whole host's containers, run one monitor instance *inside* each container (or VM) you actually want visibility into, with only the host collector enabled — `gopsutil` just reads `/proc`-level stats for wherever it's running, so it never touches the Docker socket at all. That makes it naturally scoped to exactly one target, with no socket-access question to raise.

Steps:
1. Build the binary (works from source directly — no Docker image needed for this path):
   ```bash
   go build -o infrastructure-monitor .
   ```
2. Copy `infrastructure-monitor` into whatever the target container image already builds (a `COPY` line in your own Dockerfile), or drop it directly onto a VM.
3. Run it alongside the existing process, no Docker dependency at all:
   ```bash
   SENTRY_DSN=... COLLECTORS=host ./infrastructure-monitor
   ```
4. Repeat per container or host that needs visibility. Each instance tags its metrics with its own real hostname, so multiple instances stay distinguishable in the Sentry UI without extra config.

[Security Concern](#security) for the tradeoff between Fleet-Wide and Per-Target.

## Sentry Metrics

### Source 1: Host (`gopsutil`)

Reads directly from the target's kernel via `gopsutil` — wherever this binary happens to be running (a container, a VM, bare metal). This is the per-target collector described above (`-collectors=host`) — [code](https://github.com/thinkocapo/infrastructure/blob/main/collectors/host.go#L20) where this is set.

`source` is always `gopsutil` for these metrics — it identifies which collector emitted them. `host` is the real hostname of wherever this binary is running, so multiple Per-Target instances stay distinguishable from each other in the Sentry UI.

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

Reads per-container stats from the Docker daemon. Each running container gets its own set of metrics, tagged by container name — [code](https://github.com/thinkocapo/infrastructure/blob/main/collectors/docker.go#L19) where this is set.

`source` is always `docker` for these metrics. `host` is the real hostname of the machine whose Docker daemon is being read (the Fleet-Wide instance doing the watching, not the container being watched). `container` is that container's name, as Docker reports it.

| Metric | Tags |
|---|---|
| `docker.cpu.percent` | `source`, `host`, `container` |
| `docker.memory.used_mb` | `source`, `host`, `container` |
| `docker.memory.percent` | `source`, `host`, `container` |

Docker metrics are skipped gracefully if Docker is not running.

## Sentry UI

WIP

## Security

**Docker socket access.** `-collectors=docker` (Fleet-Wide) requires mounting `/var/run/docker.sock` into the monitor's container, which is root-equivalent access to that host — a process with access to that socket can create, exec into, or mount the host filesystem into any container. Worth raising explicitly with whoever owns security for a given environment, not something to hand over quietly.

**Per-Target vs. Fleet-Wide tradeoff.** Choosing Per-Target means running one process per target instead of one process watching the whole fleet — more instances to deploy and update, in exchange for no socket exposure and no all-or-nothing visibility. `-collectors=docker` stays fully available in this same binary for local testing or any environment where the socket tradeoff is acceptable — [open a feature request](https://github.com/thinkocapo/infrastructure/issues/new) if there's something you'd like to see to make this more usable for you.