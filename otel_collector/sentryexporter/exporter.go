package sentryexporter

import (
	"context"
	"fmt"

	sentry "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// sentryExporter receives OTel metrics and forwards them to Sentry via sentry.Metrics.Gauge()
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
func (e *sentryExporter) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
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
				m := ms.At(k)
				emitMetric(m, baseTags)
			}
		}
	}
	return nil
}

func emitMetric(m pmetric.Metric, baseTags map[string]string) {
	name := m.Name()

	switch m.Type() {
	case pmetric.MetricTypeGauge:
		dps := m.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			tags := mergeTags(baseTags, dp.Attributes())
			sentry.Metrics.Gauge(name, dp.DoubleValue(), sentry.MetricTags(tags))
			fmt.Printf("  [sentry] gauge %s=%.2f tags=%v\n", name, dp.DoubleValue(), tags)
		}

	case pmetric.MetricTypeSum:
		dps := m.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			tags := mergeTags(baseTags, dp.Attributes())
			// sums (counters) mapped to gauge — Sentry metrics doesn't have a counter type
			sentry.Metrics.Gauge(name, dp.DoubleValue(), sentry.MetricTags(tags))
			fmt.Printf("  [sentry] sum->gauge %s=%.2f tags=%v\n", name, dp.DoubleValue(), tags)
		}

	default:
		// Histogram, ExponentialHistogram, Summary — skip for now
		fmt.Printf("  [sentry] skipping unsupported metric type %s for %s\n", m.Type(), name)
	}
}

// mergeTags combines resource-level attributes with datapoint-level attributes
func mergeTags(base map[string]string, attrs pcommon.Map) map[string]string {
	tags := make(map[string]string, len(base)+attrs.Len())
	for k, v := range base {
		tags[k] = v
	}
	attrs.Range(func(k string, v pcommon.Value) bool {
		tags[k] = v.AsString()
		return true
	})
	return tags
}

func (e *sentryExporter) Start(_ context.Context, _ component.Host) error  { return nil }
func (e *sentryExporter) Shutdown(_ context.Context) error                  { sentry.Flush(2 * time.Second); return nil }
func (e *sentryExporter) Capabilities() exporter.Capabilities               { return exporter.Capabilities{MutatesData: false} }
