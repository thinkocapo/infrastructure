# Benchmarking Overhead

Real numbers below, plus how to reproduce them. This benchmarking process is a maintainer task, not part of the shipped project — end users aren't expected to run it themselves, it's how these numbers get kept honest over time.

**This benchmarks the Direct SDK implementation, both run patterns: Per-Target and Fleet-Wide.** OTel Collector mode isn't benchmarked yet — it's a heavier baseline in principle (a full collector pipeline vs. a minimal purpose-built binary), but that's still unmeasured.

## How to measure it

**Per-Target** — measure the actual OS process, not the app's own claims:
```bash
go build -o infrastructure-monitor .
INTERVAL_SECONDS=5 ./infrastructure-monitor -collectors=host &
ps -o pid,%cpu,rss,vsz -p $(pgrep -f infrastructure-monitor)   # repeat a few times
```
`rss` is resident memory in KB on macOS/BSD `ps` — that's the real footprint. Ignore `vsz`; Go's large virtual address space reservation isn't actual memory use.

**Fleet-Wide** — measure the container directly:
```bash
docker compose up -d --build
docker stats --no-stream infrastructure-infra-monitor-1
```
If running behind the [Docker socket proxy](../README.md) (the recommended setup — see Security Concerns), also measure the proxy sidecar, since it's now part of the real cost of running Fleet-Wide safely:
```bash
docker stats --no-stream infrastructure-docker-proxy-1
```

## Latest measurements

Measured 2026-08-22 on a MacBook running Docker Desktop — **not yet a real cloud VM or Linux host**, that's still a separate to-do. Take these as directionally correct, not final.

| Mode | CPU | Memory |
|---|---|---|
| Per-Target (bare process, `ps`) | ~0%, brief spikes to ~1% on each collection tick | ~22–23 MB RSS |
| Fleet-Wide (`infra-monitor` container, `docker stats`) | ~0% | ~8.1 MiB |
| Docker socket proxy sidecar (`docker-proxy` container) | ~0% | ~24.9 MiB |

Two things worth noting, not hand-waving:
- The Fleet-Wide container reads *lower* memory than the bare Per-Target process above — that's the difference between a Linux cgroup memory reading (inside Docker Desktop's VM) and a macOS `ps` RSS reading, not a claim that containerizing this makes it lighter. They're not directly comparable measurements.
- Running Fleet-Wide safely (behind the socket proxy) isn't free — the proxy sidecar itself uses roughly 3x what `infra-monitor` does. Worth counting the proxy's footprint as part of Fleet-Wide's real cost, not treating it as a side effect that doesn't count.

Next real step: repeat this on an actual cloud VM (today's other to-do), under some real container load rather than idle, and update the table above with those numbers instead of these laptop ones.
