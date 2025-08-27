package tableexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// dbMetricsExporter is the component responsible for consuming and exporting metrics to Databricks.
type dbMetricsExporter struct {
	logger          *zap.Logger
	dbExporter      *databricksExporter
	targetTableName string
}

// Capabilities returns the capabilities of the metrics exporter.
func (e *dbMetricsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{
		MutatesData: false,
	}
}

// newDBMetricsExporter initializes a new instance of dbMetricsExporter.
func newDBMetricsExporter(logger *zap.Logger, dbHost, dbHttpPath, dbToken, targetTable string) (*dbMetricsExporter, error) {
	dbExporter, err := newDatabricksExporter(logger, dbHost, dbHttpPath, dbToken, targetTable)
	if err != nil {
		return nil, err
	}

	return &dbMetricsExporter{
		logger:          logger,
		dbExporter:      dbExporter,
		targetTableName: targetTable,
	}, nil
}

// Start is called to start the exporter. It is typically a no-op for this type of exporter.
func (e *dbMetricsExporter) Start(ctx context.Context, host component.Host) error {
	e.logger.Info("[dbMetricsExporter] Start called")
	return nil
}

// Shutdown is called when the exporter shuts down, typically for cleanup.
func (e *dbMetricsExporter) Shutdown(ctx context.Context) error {
	e.logger.Info("[dbMetricsExporter] Shutdown called")
	return nil
}

// ConsumeMetrics processes incoming metrics and sends them to the Databricks exporter.
func (e *dbMetricsExporter) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	e.logger.Info("[dbMetricsExporter] ConsumeMetrics called")

	if err := e.dbExporter.pushMetrics(ctx, metrics); err != nil {
		e.logger.Error("Failed to push metrics to Databricks", zap.Error(err))
		return err
	}

	return nil
}
