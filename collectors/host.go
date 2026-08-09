package collectors

import (
	"context"
	"errors"
	"fmt"

	sentry "github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
)

// CollectHost reads host metrics and ships them to Sentry. It keeps
// collecting whatever it can even if one reading fails, and returns a
// joined error describing anything that went wrong so the caller can
// report collector health.
func CollectHost(ctx context.Context) error {
	m := sentry.NewMeter(ctx)
	attrs := []attribute.Builder{
		attribute.String("source", "gopsutil"),
		attribute.String("host", HostTag()),
	}

	var errs []error

	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		m.Gauge("host.cpu.percent", percents[0], sentry.WithAttributes(attrs...))
		fmt.Printf("  [host] cpu=%.1f%%", percents[0])
	} else if err != nil {
		errs = append(errs, fmt.Errorf("cpu: %w", err))
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		m.Gauge("host.memory.used_mb", float64(vm.Used)/1024/1024, sentry.WithAttributes(attrs...))
		m.Gauge("host.memory.percent", vm.UsedPercent, sentry.WithAttributes(attrs...))
		fmt.Printf("  mem=%.1f%%", vm.UsedPercent)
	} else {
		errs = append(errs, fmt.Errorf("memory: %w", err))
	}

	if d, err := disk.Usage("/"); err == nil {
		m.Gauge("host.disk.used_gb", float64(d.Used)/1024/1024/1024, sentry.WithAttributes(attrs...))
		m.Gauge("host.disk.percent", d.UsedPercent, sentry.WithAttributes(attrs...))
		fmt.Printf("  disk=%.1f%%", d.UsedPercent)
	} else {
		errs = append(errs, fmt.Errorf("disk: %w", err))
	}

	if counters, err := gnet.IOCounters(false); err == nil && len(counters) > 0 {
		m.Gauge("host.net.bytes_sent_mb", float64(counters[0].BytesSent)/1024/1024, sentry.WithAttributes(attrs...))
		m.Gauge("host.net.bytes_recv_mb", float64(counters[0].BytesRecv)/1024/1024, sentry.WithAttributes(attrs...))
	} else if err != nil {
		errs = append(errs, fmt.Errorf("net: %w", err))
	}

	fmt.Println()
	return errors.Join(errs...)
}
