// main.go builds a custom OTel Collector binary that includes:
//   - hostmetricsreceiver  (CPU, memory, disk, network from the OS)
//   - dockerstatsreceiver  (per-container stats via Docker socket)
//   - sentryexporter       (custom — translates OTel metrics to Sentry metrics API)
package main

import (
	"log"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver"
	hostmetricsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	dockerstatsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver"
	"github.com/thinkocapo/infrastructure/otel_collector/sentryexporter"
)

func main() {
	factories, err := otelcol.Factories{
		Receivers: map[component.Type]receiver.Factory{
			component.MustNewType("hostmetrics"):  hostmetricsreceiver.NewFactory(),
			component.MustNewType("docker_stats"): dockerstatsreceiver.NewFactory(),
		},
		Exporters: map[component.Type]exporter.Factory{
			component.MustNewType("sentry"): sentryexporter.NewFactory(),
		},
	}.Build()
	if err != nil {
		log.Fatalf("failed to build factories: %v", err)
	}

	info := component.BuildInfo{
		Command:     "otelcol-sentry",
		Description: "Custom OTel Collector with Sentry metrics exporter",
		Version:     "0.1.0",
	}

	if err = otelcol.NewCommand(otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: factories,
	}).Execute(); err != nil {
		log.Fatal(err)
	}
}
