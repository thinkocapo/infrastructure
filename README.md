# infrastructure

Reads host metrics (CPU, memory, disk, network) from your MacBook and running Docker containers, then ships them to Sentry as application metrics.

## Setup

```bash
pip install -r requirements.txt
cp .env.example .env
# edit .env and add your SENTRY_DSN
```

## Run

```bash
python main.py
```

Collects and emits metrics every 60 seconds (configurable via `INTERVAL_SECONDS` in `.env`).

## Metrics emitted

| Metric | Tags |
|---|---|
| `host.cpu.percent` | `host` |
| `host.memory.used_mb` | `host` |
| `host.memory.percent` | `host` |
| `host.disk.used_gb` | `host` |
| `host.disk.percent` | `host` |
| `host.net.bytes_sent_mb` | `host` |
| `host.net.bytes_recv_mb` | `host` |
| `docker.cpu.percent` | `host`, `container` |
| `docker.memory.used_mb` | `host`, `container` |
| `docker.memory.percent` | `host`, `container` |

Docker metrics are skipped gracefully if Docker is not running.
