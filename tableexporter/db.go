package tableexporter

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	dbsql "github.com/databricks/databricks-sql-go"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// Define constant for the number of retries in case of a failed DB write
const DB_WRITE_RETRIES = 3

// time interval for table writes
const FLUSH_INTERVAL = 60 * time.Second

const MAX_CHUNK_SIZE = 50

var EndpointRegex = regexp.MustCompile(`/serving-endpoints/([^/]+)/metrics`)

// databricksExporter is responsible for exporting metrics to a Databricks table
type databricksExporter struct {
	logger        *zap.Logger
	db            *sql.DB
	mu            sync.Mutex
	metricBuffer  map[string]float64
	flushInterval time.Duration
	flushTimer    *time.Timer
	ctx           context.Context
	cancel        context.CancelFunc
	fullTableName string
}

// newDatabricksExporter initializes a new DatabricksExporter instance.
func newDatabricksExporter(logger *zap.Logger, host, httpPath, token, table string) (*databricksExporter, error) {
	connector, err := dbsql.NewConnector(
		dbsql.WithServerHostname(host),
		dbsql.WithHTTPPath(httpPath),
		dbsql.WithAccessToken(token),
		dbsql.WithTimeout(10*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %v", err)
	}

	db := sql.OpenDB(connector)
	ctx, cancel := context.WithCancel(context.Background())

	return &databricksExporter{
		logger:        logger,
		db:            db,
		metricBuffer:  make(map[string]float64),
		flushInterval: FLUSH_INTERVAL,
		ctx:           ctx,
		cancel:        cancel,
		fullTableName: table,
	}, nil
}

// pushMetrics processes and accumulates incoming metrics and stores them in the metricBuffer.
func (d *databricksExporter) pushMetrics(_ context.Context, metrics pmetric.Metrics) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tempBuffer := make(map[string]float64)

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)

		// Extract resource-level labels
		resourceAttrs := rm.Resource().Attributes()
		host := ""
		servingEndpoint := ""

		if val, ok := resourceAttrs.Get("server.address"); ok {
			host = val.AsString()
		}
		if val, ok := resourceAttrs.Get("metrics_path"); ok {
			if match := EndpointRegex.FindStringSubmatch(val.AsString()); len(match) > 1 {
				servingEndpoint = match[1]
			}
		}

		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)

			metricsSlice := sm.Metrics()
			for k := 0; k < metricsSlice.Len(); k++ {
				m := metricsSlice.At(k)

				metricName := m.Name()

				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dataPoints := m.Gauge().DataPoints()
					for dpIdx := 0; dpIdx < dataPoints.Len(); dpIdx++ {
						dp := dataPoints.At(dpIdx)
						value := getValueFromDataPoint(dp)
						key := fmt.Sprintf("%s|%s|%s", metricName, host, servingEndpoint)
						if curr, ok := tempBuffer[key]; !ok || value > curr {
							tempBuffer[key] = value
						}
					}
				case pmetric.MetricTypeSum:
					dataPoints := m.Sum().DataPoints()
					for dpIdx := 0; dpIdx < dataPoints.Len(); dpIdx++ {
						dp := dataPoints.At(dpIdx)
						value := getValueFromDataPoint(dp)
						key := fmt.Sprintf("%s|%s|%s", metricName, host, servingEndpoint)
						if curr, ok := tempBuffer[key]; !ok || value > curr {
							tempBuffer[key] = value
						}
					}
				default:
					d.logger.Debug("Skipping unsupported metric type", zap.String("metric_name", metricName))
				}
			}
		}
	}

	// After collecting all metrics, now safely update d.metricBuffer
	for metricKey, value := range tempBuffer {
		d.metricBuffer[metricKey] = value
	}

	// Start flush timer if not already started
	if d.flushTimer == nil {
		d.startFlushTimer()
	}

	return nil
}

// startFlushTimer starts a timer that will flush the metrics buffer after the set interval.
func (d *databricksExporter) startFlushTimer() {
	d.flushTimer = time.AfterFunc(d.flushInterval, func() {
		d.flush()
	})
}

// flush writes the accumulated metrics from the metricBuffer to the Databricks table.
func (d *databricksExporter) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.metricBuffer) == 0 {
		d.logger.Info("No metrics to flush.")
		return
	}

	// Prepare value placeholders and args
	var valuePlaceholders []string
	var args []interface{}

	// Convert the metricBuffer to a slice of key-value pairs for chunking
	var metricEntries []struct {
		compositeKey string
		value        float64
	}
	for compositeKey, value := range d.metricBuffer {
		metricEntries = append(metricEntries, struct {
			compositeKey string
			value        float64
		}{compositeKey, value})
	}

	// Chunk the metricEntries into blocks of 200
	for i := 0; i < len(metricEntries); i += MAX_CHUNK_SIZE {
		end := i + MAX_CHUNK_SIZE
		if end > len(metricEntries) {
			end = len(metricEntries)
		}

		// Clear the placeholders and args for this chunk
		valuePlaceholders = valuePlaceholders[:0]
		args = args[:0]

		// Process each entry in the chunk
		for j := i; j < end; j++ {
			metricEntry := metricEntries[j]
			parts := strings.Split(metricEntry.compositeKey, "|")
			if len(parts) != 3 {
				d.logger.Warn("Invalid composite key format", zap.String("key", metricEntry.compositeKey))
				continue
			}
			metricName, host, endpoint := parts[0], parts[1], parts[2]

			valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?)")
			args = append(args, metricName, host, endpoint, metricEntry.value)
		}

		query := fmt.Sprintf(`
			MERGE INTO %s AS target
			USING (
				SELECT
					col1 AS metric_name,
					col2 AS host,
					col3 AS serving_endpoint,
					col4 AS new_value
				FROM VALUES %s
			) AS source 
			ON target.metric_name = source.metric_name 
				AND target.host = source.host 
				AND target.serving_endpoint = source.serving_endpoint
			WHEN MATCHED THEN
				UPDATE SET 
					metric_values = IF(
						timestampdiff(minute, target.updated_at, current_timestamp()) >= 60,
						array_repeat(source.new_value, 60),
						IF(
							timestampdiff(minute, target.updated_at, current_timestamp()) >= 1,
							concat(
								slice(
									metric_values,
									CAST(LEAST(60, timestampdiff(minute, target.updated_at, current_timestamp())) AS INT) + 1,
									60
								),
								array_repeat(
									source.new_value,
									CAST(LEAST(60, timestampdiff(minute, target.updated_at, current_timestamp())) AS INT)
								)
							),
							metric_values
						)
					),
					updated_at = IF(
						timestampdiff(minute, target.updated_at, current_timestamp()) >= 1,
						current_timestamp(),
						target.updated_at
					)
			WHEN NOT MATCHED THEN
				INSERT (metric_name, host, serving_endpoint, metric_values, updated_at)
				VALUES (
					source.metric_name,
					source.host,
					source.serving_endpoint,
					array_repeat(source.new_value, 60),
					current_timestamp()
				)
		`, d.fullTableName, strings.Join(valuePlaceholders, ", "))

		var err error
		for attempt := 1; attempt <= DB_WRITE_RETRIES; attempt++ {
			execCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			_, err = d.db.ExecContext(execCtx, query, args...)
			cancel()

			if err == nil {
				d.logger.Info("Successfully flushed metrics chunk", zap.Int("count", len(valuePlaceholders)))
				break
			}

			d.logger.Warn("Flush failed for chunk, retrying...", zap.Int("attempt", attempt), zap.Error(err))
			time.Sleep(5 * time.Second)
		}

		if err != nil {
			d.logger.Error("Failed to flush metrics chunk after retries", zap.Error(err))
		}
	}

	// Reset buffer and timer
	d.metricBuffer = make(map[string]float64)
	d.flushTimer = nil
}

// getValueFromDataPoint converts a metric data point into a float64 value.
func getValueFromDataPoint(dp pmetric.NumberDataPoint) float64 {
	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return float64(dp.IntValue())
	case pmetric.NumberDataPointValueTypeDouble:
		return dp.DoubleValue()
	default:
		return 0
	}
}
