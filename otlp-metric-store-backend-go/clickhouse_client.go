package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// GaugeRow represents a single gauge data point for ClickHouse insertion.
type GaugeRow struct {
	ResourceAttributes    map[string]string
	ResourceSchemaUrl     string
	ScopeName             string
	ScopeVersion          string
	ScopeAttributes       map[string]string
	ScopeDroppedAttrCount uint32
	ScopeSchemaUrl        string
	ServiceName           string
	MetricName            string
	MetricDescription     string
	MetricUnit            string
	Attributes            map[string]string
	StartTimeUnix         time.Time
	TimeUnix              time.Time
	Value                 float64
	Flags                 uint32
}

// SumRow represents a single sum data point for ClickHouse insertion.
type SumRow struct {
	GaugeRow
	AggregationTemporality int32
	IsMonotonic            bool
}

// MetadataRow represents one row in otel_metrics_metadata, written once per
// distinct series rather than once per data point. MetricType (e.g. "gauge",
// "sum") is part of the MetadataID hash, so two metric types sharing an
// otherwise identical name/unit/scope/attributes don't collide on one ID.
// AggregationTemporality/IsMonotonic are nil except for Sum series.
type MetadataRow struct {
	MetadataID             uint64
	ResourceAttributes     map[string]string
	ResourceSchemaUrl      string
	ScopeName              string
	ScopeVersion           string
	ScopeAttributes        map[string]string
	ScopeDroppedAttrCount  uint32
	ScopeSchemaUrl         string
	ServiceName            string
	MetricName             string
	MetricType             string
	MetricDescription      string
	MetricUnit             string
	AggregationTemporality *int32
	IsMonotonic            *bool
	Attributes             map[string]string
	UpdatedAt              time.Time
}

// GaugeV2Row represents a single gauge data point for otel_metrics_gauge_v2,
// referencing its metadata by MetadataID instead of carrying it directly.
type GaugeV2Row struct {
	MetadataID    uint64
	StartTimeUnix time.Time
	TimeUnix      time.Time
	Value         float64
	Flags         uint32
}

// SumV2Row represents a single sum data point for otel_metrics_sum_v2.
// AggregationTemporality/IsMonotonic live on MetadataRow instead - see
// createSumV2TableSQL.
type SumV2Row struct {
	MetadataID    uint64
	StartTimeUnix time.Time
	TimeUnix      time.Time
	Value         float64
	Flags         uint32
}

// MetricsStore defines the interface for storing metrics in ClickHouse.
type MetricsStore interface {
	CreateTables(ctx context.Context) error
	InsertGauge(ctx context.Context, rows []GaugeRow) error
	InsertSum(ctx context.Context, rows []SumRow) error
	InsertMetadata(ctx context.Context, rows []MetadataRow) error
	InsertGaugeV2(ctx context.Context, rows []GaugeV2Row) error
	InsertSumV2(ctx context.Context, rows []SumV2Row) error
	Close() error
}

// ClickHouseMetricsStore implements MetricsStore using a ClickHouse connection.
type ClickHouseMetricsStore struct {
	conn driver.Conn
}

// NewClickHouseMetricsStore creates a new ClickHouseMetricsStore connected to the given address.
func NewClickHouseMetricsStore(
	ctx context.Context,
	addr string,
	database string,
	username string,
	password string,
) (*ClickHouseMetricsStore, error) {
	slog.InfoContext(ctx, "connecting to clickhouse", slog.String("addr", addr), slog.String("database", database))

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		slog.ErrorContext(ctx, "opening clickhouse connection failed", slog.String("addr", addr), slog.Any("error", err))
		return nil, fmt.Errorf("opening clickhouse connection: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		slog.ErrorContext(ctx, "pinging clickhouse failed", slog.String("addr", addr), slog.Any("error", err))
		return nil, fmt.Errorf("pinging clickhouse: %w", err)
	}

	slog.InfoContext(ctx, "connected to clickhouse", slog.String("addr", addr), slog.String("database", database))
	return &ClickHouseMetricsStore{conn: conn}, nil
}

// CreateTables executes DDL for the existing metric tables and the normalized v2 schema.
func (s *ClickHouseMetricsStore) CreateTables(ctx context.Context) error {
	slog.InfoContext(ctx, "creating tables")

	ddls := []string{
		// Existing schema - retained for backwards compatibility.
		createGaugeTableSQL,
		createSumTableSQL,
		createHistogramTableSQL,
		createExponentialHistogramTableSQL,
		createSummaryTableSQL,

		// Normalized v2 schema.
		createMetricMetadataTableSQL,
		createGaugeV2TableSQL,
		createSumV2TableSQL,
	}

	for _, ddl := range ddls {
		if err := s.conn.Exec(ctx, ddl); err != nil {
			slog.ErrorContext(ctx, "creating table failed", slog.Any("error", err))
			return fmt.Errorf("creating table: %w", err)
		}
	}

	slog.InfoContext(ctx, "tables ready", slog.Int("count", len(ddls)))
	return nil
}

// InsertGauge batch-inserts gauge rows into otel_metrics_gauge.
func (s *ClickHouseMetricsStore) InsertGauge(ctx context.Context, rows []GaugeRow) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_gauge")
	if err != nil {
		slog.ErrorContext(ctx, "preparing gauge batch failed", slog.Any("error", err))
		return fmt.Errorf("preparing gauge batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.ResourceAttributes,
			r.ResourceSchemaUrl,
			r.ScopeName,
			r.ScopeVersion,
			r.ScopeAttributes,
			r.ScopeDroppedAttrCount,
			r.ScopeSchemaUrl,
			r.ServiceName,
			r.MetricName,
			r.MetricDescription,
			r.MetricUnit,
			r.Attributes,
			r.StartTimeUnix,
			r.TimeUnix,
			r.Value,
			r.Flags,
		); err != nil {
			slog.ErrorContext(ctx, "appending gauge row failed", slog.String("metricName", r.MetricName), slog.Any("error", err))
			return fmt.Errorf("appending gauge row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		slog.ErrorContext(ctx, "sending gauge batch failed", slog.Int("rows", len(rows)), slog.Any("error", err))
		return fmt.Errorf("sending gauge batch: %w", err)
	}
	return nil
}

// InsertSum batch-inserts sum rows into otel_metrics_sum.
func (s *ClickHouseMetricsStore) InsertSum(ctx context.Context, rows []SumRow) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_sum")
	if err != nil {
		slog.ErrorContext(ctx, "preparing sum batch failed", slog.Any("error", err))
		return fmt.Errorf("preparing sum batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.ResourceAttributes,
			r.ResourceSchemaUrl,
			r.ScopeName,
			r.ScopeVersion,
			r.ScopeAttributes,
			r.ScopeDroppedAttrCount,
			r.ScopeSchemaUrl,
			r.ServiceName,
			r.MetricName,
			r.MetricDescription,
			r.MetricUnit,
			r.Attributes,
			r.StartTimeUnix,
			r.TimeUnix,
			r.Value,
			r.Flags,
			r.AggregationTemporality,
			r.IsMonotonic,
		); err != nil {
			slog.ErrorContext(ctx, "appending sum row failed", slog.String("metricName", r.MetricName), slog.Any("error", err))
			return fmt.Errorf("appending sum row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		slog.ErrorContext(ctx, "sending sum batch failed", slog.Int("rows", len(rows)), slog.Any("error", err))
		return fmt.Errorf("sending sum batch: %w", err)
	}
	return nil
}

// InsertMetadata batch-inserts metadata rows into otel_metrics_metadata.
func (s *ClickHouseMetricsStore) InsertMetadata(ctx context.Context, rows []MetadataRow) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_metadata")
	if err != nil {
		slog.ErrorContext(ctx, "preparing metadata batch failed", slog.Any("error", err))
		return fmt.Errorf("preparing metadata batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.MetadataID,
			r.ResourceAttributes,
			r.ResourceSchemaUrl,
			r.ScopeName,
			r.ScopeVersion,
			r.ScopeAttributes,
			r.ScopeDroppedAttrCount,
			r.ScopeSchemaUrl,
			r.ServiceName,
			r.MetricName,
			r.MetricType,
			r.MetricDescription,
			r.MetricUnit,
			r.AggregationTemporality,
			r.IsMonotonic,
			r.Attributes,
			r.UpdatedAt,
		); err != nil {
			slog.ErrorContext(ctx, "appending metadata row failed", slog.Uint64("metadataID", r.MetadataID), slog.Any("error", err))
			return fmt.Errorf("appending metadata row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		slog.ErrorContext(ctx, "sending metadata batch failed", slog.Int("rows", len(rows)), slog.Any("error", err))
		return fmt.Errorf("sending metadata batch: %w", err)
	}
	return nil
}

// InsertGaugeV2 batch-inserts gauge rows into otel_metrics_gauge_v2.
func (s *ClickHouseMetricsStore) InsertGaugeV2(ctx context.Context, rows []GaugeV2Row) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_gauge_v2")
	if err != nil {
		slog.ErrorContext(ctx, "preparing gauge_v2 batch failed", slog.Any("error", err))
		return fmt.Errorf("preparing gauge_v2 batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.MetadataID,
			r.StartTimeUnix,
			r.TimeUnix,
			r.Value,
			r.Flags,
		); err != nil {
			slog.ErrorContext(ctx, "appending gauge_v2 row failed", slog.Uint64("metadataID", r.MetadataID), slog.Any("error", err))
			return fmt.Errorf("appending gauge_v2 row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		slog.ErrorContext(ctx, "sending gauge_v2 batch failed", slog.Int("rows", len(rows)), slog.Any("error", err))
		return fmt.Errorf("sending gauge_v2 batch: %w", err)
	}
	return nil
}

// InsertSumV2 batch-inserts sum rows into otel_metrics_sum_v2.
func (s *ClickHouseMetricsStore) InsertSumV2(ctx context.Context, rows []SumV2Row) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_sum_v2")
	if err != nil {
		slog.ErrorContext(ctx, "preparing sum_v2 batch failed", slog.Any("error", err))
		return fmt.Errorf("preparing sum_v2 batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.MetadataID,
			r.StartTimeUnix,
			r.TimeUnix,
			r.Value,
			r.Flags,
		); err != nil {
			slog.ErrorContext(ctx, "appending sum_v2 row failed", slog.Uint64("metadataID", r.MetadataID), slog.Any("error", err))
			return fmt.Errorf("appending sum_v2 row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		slog.ErrorContext(ctx, "sending sum_v2 batch failed", slog.Int("rows", len(rows)), slog.Any("error", err))
		return fmt.Errorf("sending sum_v2 batch: %w", err)
	}
	return nil
}

// Close closes the underlying ClickHouse connection.
func (s *ClickHouseMetricsStore) Close() error {
	return s.conn.Close()
}
