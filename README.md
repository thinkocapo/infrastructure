# infrastructure

<img src="sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

Collects host metrics (CPU, memory, disk, network) and Docker container stats, then ships them to Sentry as application metrics — using the Sentry SDK directly, with no separate monitoring agent to deploy. See `collectors/` for how it's done via `gopsutil` and the Docker Engine API.

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
go run . -collectors=host          # host metrics only
go run . -collectors=docker        # Docker container metrics only
go run . -collectors=host,docker   # both (same as default)
go run .                           # all registered collectors
```

## Production deployment: per-target vs. fleet-wide

`docker-compose.yml` and `-collectors=docker` above are for trying this repo's example locally — they're not yet the right default for pointing this at a customer's real production containers. There are two genuinely different ways to deploy this, with a real tradeoff between them:

### Fleet-wide (`-collectors=docker`) — good for demos, not yet for production

One monitor instance per host, talking to that host's Docker socket, reporting on every container the daemon can see. This is what `docker-compose.yml` does. Two things to know before pointing this at a real environment:
- It has no way to scope to "just these containers" — it reports on the whole daemon, all or nothing.
- It requires mounting `/var/run/docker.sock` into the container, which is root-equivalent access to that host — worth raising explicitly with whoever owns security for that environment, not something to hand over quietly.

### Per-target (`-collectors=host`) — recommended for production today

Instead of one monitor watching a whole host's containers, run one monitor instance *inside* each container (or VM) you actually want visibility into, with only the host collector enabled. It never touches the Docker socket — `gopsutil` just reads `/proc`-level stats for wherever it's running — so it's naturally scoped to exactly one target, and there's no socket-access question to raise at all.

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

The tradeoff to be upfront about: this trades "one process watches the whole fleet" for "one process per target" — more instances to deploy and update, in exchange for no socket exposure and no all-or-nothing visibility. `-collectors=docker` stays fully available in this same binary for local testing or any environment where the socket tradeoff is acceptable — nothing about recommending the per-target pattern removes or restricts it.

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

In the Sentry metrics explorer, use the `source` tag to filter or group by where metrics came from. Example tag schemes as more collectors are added:

```go
sentry.Metrics.Gauge("host.cpu.percent",   value, sentry.MetricTags({"source": "gopsutil",   "host": "web-01"}))
sentry.Metrics.Gauge("docker.cpu.percent", value, sentry.MetricTags({"source": "docker",     "container": "postgres"}))
sentry.Metrics.Gauge("k8s.pod.memory",     value, sentry.MetricTags({"source": "kubernetes", "namespace": "default"}))
```

In the UI: filter by `source = docker` to see only container metrics, or group by `container` to compare across containers. The `source` tag is the top-level discriminator; more specific tags (`host`, `container`, `namespace`) let you drill down within a source.