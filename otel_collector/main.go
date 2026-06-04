// main.go builds a custom OTel Collector binary that includes:
//   - hostmetricsreceiver  (CPU, memory, disk, network from the OS)
//   - dockerstatsreceiver  (per-container stats via Docker socket)
//   - sentryexporter       (custom — translates OTel metrics to Sentry metrics API)
//
// Metric names emitted use OTel standard naming, which differs from direct SDK mode:
//   - hostmetricsreceiver: system.cpu.utilization, system.memory.usage, system.disk.io, etc.
//   - dockerstatsreceiver: container.cpu.utilization, container.memory.usage, etc.
package main

import (
	"log"

	dockerstatsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver"
	hostmetricsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/thinkocapo/infrastructure/otel_collector/sentryexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver"
)

func main() {
	factories := otelcol.Factories{
		Receivers: map[component.Type]receiver.Factory{
			component.MustNewType("hostmetrics"):  hostmetricsreceiver.NewFactory(),
			component.MustNewType("docker_stats"): dockerstatsreceiver.NewFactory(),
		},
		Exporters: map[component.Type]exporter.Factory{
			component.MustNewType("sentry"): sentryexporter.NewFactory(),
		},
	}

	info := component.BuildInfo{
		Command:     "otelcol-sentry",
		Description: "Custom OTel Collector with Sentry metrics exporter",
		Version:     "0.1.0",
	}

	if err := otelcol.NewCommand(otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: func() (otelcol.Factories, error) { return factories, nil },
	}).Execute(); err != nil {
		log.Fatal(err)
	}
}
