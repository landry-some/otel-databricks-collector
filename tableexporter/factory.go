package tableexporter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const typeStr = "tableexporter"

// NewFactory creates a factory for the custom metrics exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelAlpha),
	)
}

// createDefaultConfig creates the default configuration for the metrics exporter.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createMetricsExporter creates a new dbMetricsExporter and wraps it in a metrics exporter.
func createMetricsExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {

	cfg := config.(*Config)
	dbexp, err := newDBMetricsExporter(set.Logger, cfg.DBHost, cfg.DBHttpPath, cfg.DBToken, cfg.TargetTable)
	if err != nil {
		return nil, fmt.Errorf("failed to create dbMetricsExporter: %w", err)
	}
	return exporterhelper.NewMetrics(ctx, set, config, dbexp.ConsumeMetrics)
}
