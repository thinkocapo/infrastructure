# Security Concerns

**Docker socket access.** `-collectors=docker` (Fleet-Wide) requires mounting `/var/run/docker.sock` into the monitor's container, which is root-equivalent access to that host — a process with access to that socket can create, exec into, or mount the host filesystem into any container. Worth raising explicitly with whoever owns security for a given environment, not something to hand over quietly.

**Per-Target vs. Fleet-Wide tradeoff.** Choosing Per-Target means running one process per target instead of one process watching the whole fleet — more instances to deploy and update, in exchange for no socket exposure and no all-or-nothing visibility. `-collectors=docker` stays fully available in this same binary for local testing or any environment where the socket tradeoff is acceptable — [open a feature request](https://github.com/thinkocapo/infrastructure/issues/new) if there's something you'd like to see to make this more usable for you.

**No retry/backoff on the OTel exporter.** This is a shrug to a developer and a real concern to an SRE — silent data loss during a transient network blip is specifically the kind of failure mode SREs get paged for and design against professionally. Known gap, tracked in the Handbook's Development section — may be fixed soon.

**Container scoping as a data-governance question, not just a feature gap.** Fleet-Wide's "no way to scope to specific containers" means that on a shared multi-tenant host, you're reporting metrics — and container names — for another team's workload without their knowledge. Worth a real conversation with whoever owns that host before running Fleet-Wide there.
