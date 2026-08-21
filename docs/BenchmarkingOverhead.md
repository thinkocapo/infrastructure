# Benchmarking Overhead

Not formally benchmarked yet with real numbers — that's on the roadmap (see Development in the Handbook), and the plan is to measure it on an actual cloud VM rather than guess. Qualitatively, here's the shape:

- **Per-Target / Direct SDK:** a small, statically-built Go binary doing a handful of `gopsutil` reads (CPU, memory, disk, network) once per interval. No runtime to speak of — no JVM, no interpreter — so the floor is very low, in the same ballpark as any lightweight Go CLI tool idling between ticks.
- **Fleet-Wide / Direct SDK:** the same, plus one Docker Engine API call per container per interval. Scales linearly with container count on that host.
- **OTel Collector mode:** a heavier baseline than the direct SDK binary, since it's a full OTel Collector process (pipeline framework, receivers, batching) rather than a minimal purpose-built binary — how much heavier isn't measured yet.

Once real numbers exist (CPU/memory of the `infra-monitor` container or process itself, ideally from a real VM rather than a laptop), they belong here instead of this qualitative description.
