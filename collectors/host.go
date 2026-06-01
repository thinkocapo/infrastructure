package collectors

import (
	"context"
	"fmt"

	sentry "github.com/getsentry/sentry-go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func CollectHost(ctx context.Context) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	tags := map[string]string{"source": "psutil", "host": "macbook"}

	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		hub.Scope().SetTag("source", "psutil")
		sentry.Metrics.Gauge("host.cpu.percent", percents[0], sentry.MetricTags(tags))
		fmt.Printf("  [host] cpu=%.1f%%", percents[0])
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		usedMB := float64(vm.Used) / 1024 / 1024
		sentry.Metrics.Gauge("host.memory.used_mb", usedMB, sentry.MetricTags(tags))
		sentry.Metrics.Gauge("host.memory.percent", vm.UsedPercent, sentry.MetricTags(tags))
		fmt.Printf("  mem=%.1f%%", vm.UsedPercent)
	}

	if d, err := disk.Usage("/"); err == nil {
		usedGB := float64(d.Used) / 1024 / 1024 / 1024
		sentry.Metrics.Gauge("host.disk.used_gb", usedGB, sentry.MetricTags(tags))
		sentry.Metrics.Gauge("host.disk.percent", d.UsedPercent, sentry.MetricTags(tags))
		fmt.Printf("  disk=%.1f%%", d.UsedPercent)
	}

	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		sentMB := float64(counters[0].BytesSent) / 1024 / 1024
		recvMB := float64(counters[0].BytesRecv) / 1024 / 1024
		sentry.Metrics.Gauge("host.net.bytes_sent_mb", sentMB, sentry.MetricTags(tags))
		sentry.Metrics.Gauge("host.net.bytes_recv_mb", recvMB, sentry.MetricTags(tags))
	}

	fmt.Println()
}
