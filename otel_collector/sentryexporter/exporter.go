package sentryexporter

import (
	"context"
	"fmt"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// sentryExporter receives OTel metrics and forwards them to Sentry via the metrics API.
// Metric names coming from OTel receivers use OTel standard naming:
//   - hostmetricsreceiver: system.cpu.utilization, system.memory.usage, system.disk.io, etc.
//   - dockerstatsreceiver: container.cpu.utilization, container.memory.usage, etc.
// These are distinct from the direct SDK mode names (host.cpu.percent, docker.cpu.percent).
type sentryExporter struct {
	dsn string
}

func newSentryExporter(cfg *Config) (*sentryExporter, error) {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		TracesSampleRate: 0.0,
	}); err != nil {
		return nil, fmt.Errorf("sentry.Init: %w", err)
	}
	return &sentryExporter{dsn: cfg.DSN}, nil
}

// ConsumeMetrics is called by the OTel Collector for each batch of metrics.
// It walks the OTel metrics tree and emits each data point as a Sentry gauge.
func (e *sentryExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	m := sentry.NewMeter(ctx)

	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		// pull resource-level attributes (e.g. host.name) as base tags
		baseTags := map[string]string{}
		rm.Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
			baseTags[k] = v.AsString()
			return true
		})

		sms := rm.ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			ms := sms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				emitMetric(m, ms.At(k), baseTags)
			}
		}
	}
	return nil
}

func emitMetric(m sentry.Meter, metric pmetric.Metric, baseTags map[string]string) {
	name := metric.Name()

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			attrs := buildAttrs(baseTags, dp.Attributes())
			m.Gauge(name, dp.DoubleValue(), sentry.WithAttributes(attrs...))
			fmt.Printf("  [otel→sentry] gauge %s=%.2f\n", name, dp.DoubleValue())
		}

	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			attrs := buildAttrs(baseTags, dp.Attributes())
			// Sentry has no counter type — map sums to gauge
			m.Gauge(name, dp.DoubleValue(), sentry.WithAttributes(attrs...))
			fmt.Printf("  [otel→sentry] sum→gauge %s=%.2f\n", name, dp.DoubleValue())
		}

	default:
		// Histogram, ExponentialHistogram, Summary — skip for now
		fmt.Printf("  [otel→sentry] skipping unsupported type %s for %s\n", metric.Type(), name)
	}
}

// buildAttrs merges resource-level and datapoint-level attributes into OTel attribute builders.
func buildAttrs(base map[string]string, dp pcommon.Map) []attribute.Builder {
	attrs := make([]attribute.Builder, 0, len(base)+dp.Len())
	for k, v := range base {
		attrs = append(attrs, attribute.String(k, v))
	}
	dp.Range(func(k string, v pcommon.Value) bool {
		attrs = append(attrs, attribute.String(k, v.AsString()))
		return true
	})
	return attrs
}

func (e *sentryExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (e *sentryExporter) Shutdown(_ context.Context) error {
	sentry.Flush(2 * time.Second)
	return nil
}

func (e *sentryExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
