# infrastructure

Reads host metrics (CPU, memory, disk, network) from your MacBook and running Docker containers, then ships them to Sentry as application metrics.

The pattern for adding more sources (Kubernetes, Postgres, Redis, etc.) is just another `collect_X_metrics()` function in `main.py` — same loop, same Sentry emit calls. One Sentry project and DSN is enough; differentiate sources with tags.

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

## Scheduling

The script needs to run continuously (or be invoked repeatedly) to keep shipping metrics. Four options:

### 1. Built-in loop (default, recommended for demos)

The script already contains a `while True` loop with a configurable sleep. Just run it once and leave it:

```bash
python main.py
```

Set `INTERVAL_SECONDS=60` in `.env` to control the cadence. Kill with `Ctrl+C`.

### 2. cron

Invokes the script as a new process on a schedule. Simple, but each run is independent — no persistent state.

```bash
crontab -e
# add:
* * * * * /usr/bin/python3 /Users/you/thinkocapo/infrastructure/main.py
```

Note: if using this approach, remove the `while True` loop so each cron invocation does a single collection pass.

### 3. launchd (macOS native, production-grade)

macOS's built-in job scheduler. More robust than cron — supports auto-restart on failure, logging, and boot-time launch. Create a plist at `~/Library/LaunchAgents/com.thinkocapo.infrastructure.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.thinkocapo.infrastructure</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/python3</string>
        <string>/Users/you/thinkocapo/infrastructure/main.py</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>StandardOutPath</key>
    <string>/tmp/infrastructure.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/infrastructure.err</string>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

Load it with:
```bash
launchctl load ~/Library/LaunchAgents/com.thinkocapo.infrastructure.plist
```

### 4. Docker (self-contained, portable)

Run the script in a container with the built-in loop. Docker Desktop on Mac can pass through `/var/run/docker.sock` for Docker-in-Docker container stats:

```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
CMD ["python", "main.py"]
```

```bash
docker build -t infrastructure-monitor .
docker run -d \
  --env-file .env \
  -v /var/run/docker.sock:/var/run/docker.sock \
  infrastructure-monitor
```

---

## Metrics emitted

### Source 1: macOS host (`psutil`)

Reads directly from the macOS kernel — no dependencies beyond `psutil`.

| Metric | Tags |
|---|---|
| `host.cpu.percent` | `host` |
| `host.memory.used_mb` | `host` |
| `host.memory.percent` | `host` |
| `host.disk.used_gb` | `host` |
| `host.disk.percent` | `host` |
| `host.net.bytes_sent_mb` | `host` |
| `host.net.bytes_recv_mb` | `host` |

### Source 2: Docker containers (Docker Engine API)

Reads per-container stats from the Docker daemon. Each running container gets its own set of metrics, tagged by container name.

| Metric | Tags |
|---|---|
| `docker.cpu.percent` | `host`, `container` |
| `docker.memory.used_mb` | `host`, `container` |
| `docker.memory.percent` | `host`, `container` |

Docker metrics are skipped gracefully if Docker is not running or the `docker` SDK is not installed.

## Querying by source in the Sentry UI

In the Sentry metrics explorer, use the `source` tag to filter or group by where metrics came from. Example tag schemes as more collectors are added:

```python
metrics.gauge("host.cpu.percent",   value, tags={"source": "psutil",     "host": "macbook"})
metrics.gauge("docker.cpu.percent", value, tags={"source": "docker",     "container": "postgres"})
metrics.gauge("k8s.pod.memory",     value, tags={"source": "kubernetes", "namespace": "default"})
```

In the UI: filter by `source = docker` to see only container metrics, or group by `container` to compare across containers. The `source` tag is the top-level discriminator; more specific tags (`host`, `container`, `namespace`) let you drill down within a source.
