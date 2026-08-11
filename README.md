# infrastructure

<img src="sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

Collects host metrics (CPU, memory, disk, network) and Docker container stats, then ships them to Sentry as application metrics — using the Sentry SDK directly, with no separate monitoring agent to deploy. See `collectors/` for how it's done via `gopsutil` and the Docker Engine API.

## Setup
```bash
# you'll need a SENTRY_DSN to add to .env below
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

### Run Per-Target

Instead of one monitor watching a whole host's containers, run one monitor instance *inside* each container (or VM) you actually want visibility into, with only the host collector enabled (`-collectors=host`). It never touches the Docker socket — `gopsutil` just reads `/proc`-level stats for wherever it's running — so it's naturally scoped to exactly one target, and there's no socket-access question to raise at all.

Steps:
1. Build the binary (works from source directly — no Docker image needed for this path):
   ```bash
   go build -o infrastructure-monitor .
   ```
2. Copy `infrastructure-monitor` into whatever the target container image already builds (a `COPY` line in the customer's own Dockerfile), or drop it directly onto a VM.
3. Run it alongside the existing process, no Docker dependency at all:
   ```bash
   SENTRY_DSN=... COLLECTORS=host ./infrastructure-monitor
   ```
4. Repeat per container or host that needs visibility. Each instance tags its metrics with its own real hostname, so multiple instances stay distinguishable in the Sentry UI without extra config.

See [Security Concern](#security) for the tradeoff between this pattern and Fleet-Wide.

### Run Fleet-Wide

One monitor instance per host, run with `-collectors=docker`, talking to that host's Docker socket and reporting on every container the daemon can see — exactly what the Demo App above does, via `docker-compose.yml`. To run it standalone, outside of Compose:

```bash
docker build -t infrastructure-monitor .
docker run -d --restart unless-stopped \
  --env-file .env \
  -e COLLECTORS=docker \
  -v /var/run/docker.sock:/var/run/docker.sock \
  infrastructure-monitor
```

Two things to know before pointing this at a real environment:
- It has no way to scope to "just these containers" — it reports on the whole daemon, all or nothing. [Open a GitHub Issue](https://github.com/thinkocapo/infrastructure/issues/new) if you have a feature request.
- It requires mounting `/var/run/docker.sock` into the container, which is root-equivalent access to that host — worth raising explicitly with whoever owns security for that environment, not something to hand over quietly.

## Sentry Metrics

### Source 1: Host (`gopsutil`)

Reads directly from the target's kernel via `gopsutil` — wherever this binary happens to be running (a container, a VM, bare metal). This is the per-target collector described above (`-collectors=host`).

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

## Sentry UI

WIP

## Security

**Docker socket access.** `-collectors=docker` (Fleet-Wide) requires mounting `/var/run/docker.sock` into the monitor's container, which is root-equivalent access to that host — a process with access to that socket can create, exec into, or mount the host filesystem into any container. Worth raising explicitly with whoever owns security for a given environment, not something to hand over quietly.

**Per-Target vs. Fleet-Wide tradeoff.** This trades "one process watches the whole fleet" for "one process per target" — more instances to deploy and update, in exchange for no socket exposure and no all-or-nothing visibility. `-collectors=docker` stays fully available in this same binary for local testing or any environment where the socket tradeoff is acceptable — nothing about recommending the per-target pattern removes or restricts it.

If you'd like this separated out as its own feature (e.g. a container allow-list so Fleet-Wide isn't all-or-nothing, or a read-only socket proxy in front of the Docker socket), [open a feature request](https://github.com/thinkocapo/infrastructure/issues/new).