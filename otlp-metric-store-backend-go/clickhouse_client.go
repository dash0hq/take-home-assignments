package main

import (
	"context"
	"fmt"
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

// MetricsStore defines the interface for storing metrics in ClickHouse.
type MetricsStore interface {
	CreateTables(ctx context.Context) error
	InsertGauge(ctx context.Context, rows []GaugeRow) error
	InsertSum(ctx context.Context, rows []SumRow) error
	Close() error
}

// ClickHouseMetricsStore implements MetricsStore using a ClickHouse connection.
type ClickHouseMetricsStore struct {
	conn driver.Conn
}

// NewClickHouseMetricsStore creates a new ClickHouseMetricsStore connected to the given address.
func NewClickHouseMetricsStore(ctx context.Context, addr string, database string, username string, password string) (*ClickHouseMetricsStore, error) {
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
		return nil, fmt.Errorf("opening clickhouse connection: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("pinging clickhouse: %w", err)
	}
	return &ClickHouseMetricsStore{conn: conn}, nil
}

// CreateTables executes DDL for all 5 metric tables.
func (s *ClickHouseMetricsStore) CreateTables(ctx context.Context) error {
	ddls := []string{
		createGaugeTableSQL,
		createSumTableSQL,
		createHistogramTableSQL,
		createExponentialHistogramTableSQL,
		createSummaryTableSQL,
	}
	for _, ddl := range ddls {
		if err := s.conn.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("creating table: %w", err)
		}
	}
	return nil
}

// InsertGauge batch-inserts gauge rows into otel_metrics_gauge.
func (s *ClickHouseMetricsStore) InsertGauge(ctx context.Context, rows []GaugeRow) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_gauge")
	if err != nil {
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
			return fmt.Errorf("appending gauge row: %w", err)
		}
	}
	return batch.Send()
}

// InsertSum batch-inserts sum rows into otel_metrics_sum.
func (s *ClickHouseMetricsStore) InsertSum(ctx context.Context, rows []SumRow) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO otel_metrics_sum")
	if err != nil {
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
			return fmt.Errorf("appending sum row: %w", err)
		}
	}
	return batch.Send()
}

// Close closes the underlying ClickHouse connection.
func (s *ClickHouseMetricsStore) Close() error {
	return s.conn.Close()
}
