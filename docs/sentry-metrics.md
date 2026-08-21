## Sentry Metrics

<img src="../sentry_metrics.png" width="50%" alt="Sentry infrastructure metrics dashboard">

### Source 1: Host (`gopsutil`)

Reads directly from the target's kernel via `gopsutil` — wherever this binary happens to be running (a container, a VM, bare metal). This is the per-target collector described in the root README's Run section (`-collectors=host`) — [code](https://github.com/thinkocapo/infrastructure/blob/main/collectors/host.go#L20) where this is set.

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
