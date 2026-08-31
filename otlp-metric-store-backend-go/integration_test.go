//go:build integration

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func setupClickHouse(t *testing.T) (*ClickHouseMetricsStore, func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := testcontainers.Run(ctx, "clickhouse/clickhouse-server:26.2",
		testcontainers.WithExposedPorts("9000/tcp"),
		testcontainers.WithEnv(map[string]string{
			"CLICKHOUSE_USER":     "default",
			"CLICKHOUSE_PASSWORD": "test",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("9000/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("starting clickhouse container: %v", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("getting container host: %v", err)
	}
	mappedPort, err := ctr.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatalf("getting mapped port: %v", err)
	}

	addr := fmt.Sprintf("%s:%s", host, mappedPort.Port())
	store, err := NewClickHouseMetricsStore(ctx, addr, "default", "default", "test")
	if err != nil {
		t.Fatalf("creating clickhouse metrics store: %v", err)
	}

	cleanup := func() {
		store.Close()
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminating clickhouse container: %v", err)
		}
	}

	return store, cleanup
}

// TestCreateTables verifies CreateTables creates all 8 expected tables.
func TestCreateTables(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	expectedTables := []string{
		"otel_metrics_gauge",
		"otel_metrics_sum",
		"otel_metrics_histogram",
		"otel_metrics_exponential_histogram",
		"otel_metrics_summary",

		// New normalized schema.
		"otel_metrics_metadata",
		"otel_metrics_gauge_v2",
		"otel_metrics_sum_v2",
	}

	for _, table := range expectedTables {
		var count uint64
		err := store.conn.QueryRow(ctx,
			"SELECT count() FROM system.tables WHERE database = 'default' AND name = $1", table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("querying system.tables for %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist, got count=%d", table, count)
		}
	}
}

// TestInsertGauge verifies a mapped Gauge row round-trips through the legacy otel_metrics_gauge table.
func TestInsertGauge(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	now := uint64(time.Now().UnixNano())
	startTime := now - uint64(time.Minute)
	resourceMetrics := []*metricspb.ResourceMetrics{
		{
			Resource: &resourcepb.Resource{
				Attributes: []*commonpb.KeyValue{
					{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "test-service"}}},
					{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "test-host"}}},
				},
			},
			SchemaUrl: "https://opentelemetry.io/schemas/1.4.0",
			ScopeMetrics: []*metricspb.ScopeMetrics{
				{
					Scope: &commonpb.InstrumentationScope{
						Name:    "test-scope",
						Version: "1.0.0",
					},
					Metrics: []*metricspb.Metric{
						{
							Name:        "cpu.utilization",
							Description: "CPU utilization percentage",
							Unit:        "%",
							Data: &metricspb.Metric_Gauge{
								Gauge: &metricspb.Gauge{
									DataPoints: []*metricspb.NumberDataPoint{
										{
											Attributes:        []*commonpb.KeyValue{{Key: "cpu", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "0"}}}},
											StartTimeUnixNano: startTime,
											TimeUnixNano:      now,
											Value:             &metricspb.NumberDataPoint_AsDouble{AsDouble: 42.5},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	rows := MapGaugeRows(resourceMetrics)
	if err := store.InsertGauge(ctx, rows); err != nil {
		t.Fatalf("inserting gauge rows: %v", err)
	}

	var (
		serviceName string
		metricName  string
		value       float64
	)
	err := store.conn.QueryRow(ctx,
		"SELECT ServiceName, MetricName, Value FROM otel_metrics_gauge WHERE MetricName = 'cpu.utilization'",
	).Scan(&serviceName, &metricName, &value)
	if err != nil {
		t.Fatalf("querying gauge: %v", err)
	}

	if serviceName != "test-service" {
		t.Errorf("expected ServiceName=test-service, got %s", serviceName)
	}
	if metricName != "cpu.utilization" {
		t.Errorf("expected MetricName=cpu.utilization, got %s", metricName)
	}
	if value != 42.5 {
		t.Errorf("expected Value=42.5, got %f", value)
	}
}

// TestInsertSum verifies a mapped Sum row, including AggregationTemporality/IsMonotonic, round-trips through otel_metrics_sum.
func TestInsertSum(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	now := uint64(time.Now().UnixNano())
	startTime := now - uint64(time.Minute)
	resourceMetrics := []*metricspb.ResourceMetrics{
		{
			Resource: &resourcepb.Resource{
				Attributes: []*commonpb.KeyValue{
					{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "test-service"}}},
					{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "test-host"}}},
				},
			},
			SchemaUrl: "https://opentelemetry.io/schemas/1.4.0",
			ScopeMetrics: []*metricspb.ScopeMetrics{
				{
					Scope: &commonpb.InstrumentationScope{
						Name:    "test-scope",
						Version: "1.0.0",
					},
					Metrics: []*metricspb.Metric{
						{
							Name:        "http.requests.total",
							Description: "Total HTTP requests",
							Unit:        "{request}",
							Data: &metricspb.Metric_Sum{
								Sum: &metricspb.Sum{
									AggregationTemporality: metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
									IsMonotonic:            true,
									DataPoints: []*metricspb.NumberDataPoint{
										{
											Attributes: []*commonpb.KeyValue{
												{Key: "method", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "GET"}}},
												{Key: "status", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "200"}}},
											},
											StartTimeUnixNano: startTime,
											TimeUnixNano:      now,
											Value:             &metricspb.NumberDataPoint_AsDouble{AsDouble: 1234},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	rows := MapSumRows(resourceMetrics)
	if err := store.InsertSum(ctx, rows); err != nil {
		t.Fatalf("inserting sum rows: %v", err)
	}

	var (
		serviceName            string
		metricName             string
		value                  float64
		aggregationTemporality int32
		isMonotonic            bool
	)
	err := store.conn.QueryRow(ctx,
		"SELECT ServiceName, MetricName, Value, AggregationTemporality, IsMonotonic FROM otel_metrics_sum WHERE MetricName = 'http.requests.total'",
	).Scan(&serviceName, &metricName, &value, &aggregationTemporality, &isMonotonic)
	if err != nil {
		t.Fatalf("querying sum: %v", err)
	}

	if serviceName != "test-service" {
		t.Errorf("expected ServiceName=test-service, got %s", serviceName)
	}
	if metricName != "http.requests.total" {
		t.Errorf("expected MetricName=http.requests.total, got %s", metricName)
	}
	if value != 1234 {
		t.Errorf("expected Value=1234, got %f", value)
	}
	if aggregationTemporality != 2 {
		t.Errorf("expected AggregationTemporality=2, got %d", aggregationTemporality)
	}
	if !isMonotonic {
		t.Errorf("expected IsMonotonic=true, got false")
	}
}

// TestCreateTables_IdempotentOnRestart verifies CreateTables tolerates
// running again against a database that already has all 8 tables, since it
// runs on every server startup.
func TestCreateTables_IdempotentOnRestart(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("first CreateTables: %v", err)
	}
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("second CreateTables (idempotency) failed: %v", err)
	}
}

// TestInsertMetadata verifies a MetadataRow, including populated
// AggregationTemporality/IsMonotonic, round-trips through otel_metrics_metadata.
func TestInsertMetadata(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	resAttrs := map[string]string{"service.name": "meta-test-service", "host.name": "meta-host"}
	scopeAttrs := map[string]string{"lib": "otel-go"}
	dpAttrs := map[string]string{"cpu": "0", "state": "user"}
	id := computeMetadataID(canonicalMetadataKey("meta.cpu.util", metricTypeSum, "1", "meta-scope", "1.0.0", resAttrs, scopeAttrs, dpAttrs))

	aggTemporality := int32(metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
	isMonotonic := true
	row := MetadataRow{
		MetadataID:             id,
		ResourceAttributes:     resAttrs,
		ResourceSchemaUrl:      "https://opentelemetry.io/schemas/1.4.0",
		ScopeName:              "meta-scope",
		ScopeVersion:           "1.0.0",
		ScopeAttributes:        scopeAttrs,
		ScopeDroppedAttrCount:  0,
		ScopeSchemaUrl:         "https://opentelemetry.io/schemas/1.4.0",
		ServiceName:            "meta-test-service",
		MetricName:             "meta.cpu.util",
		MetricType:             metricTypeSum,
		MetricDescription:      "CPU utilization for metadata test",
		MetricUnit:             "1",
		AggregationTemporality: &aggTemporality,
		IsMonotonic:            &isMonotonic,
		Attributes:             dpAttrs,
		UpdatedAt:              time.Now().UTC(),
	}
	if err := store.InsertMetadata(ctx, []MetadataRow{row}); err != nil {
		t.Fatalf("inserting metadata row: %v", err)
	}

	var (
		gotID          uint64
		gotServiceName string
		gotType        string
		gotUnit        string
		gotTemporality int32
		gotIsMonotonic bool
	)
	err := store.conn.QueryRow(ctx,
		"SELECT MetadataID, ServiceName, MetricType, MetricUnit, AggregationTemporality, IsMonotonic FROM otel_metrics_metadata WHERE MetricName = 'meta.cpu.util'",
	).Scan(&gotID, &gotServiceName, &gotType, &gotUnit, &gotTemporality, &gotIsMonotonic)
	if err != nil {
		t.Fatalf("querying metadata: %v", err)
	}
	if gotID != id {
		t.Errorf("expected MetadataID=%d, got %d", id, gotID)
	}
	if gotServiceName != "meta-test-service" {
		t.Errorf("expected ServiceName=meta-test-service, got %s", gotServiceName)
	}
	if gotType != metricTypeSum {
		t.Errorf("expected MetricType=%s, got %s", metricTypeSum, gotType)
	}
	if gotUnit != "1" {
		t.Errorf("expected MetricUnit=1, got %s", gotUnit)
	}
	if gotTemporality != aggTemporality {
		t.Errorf("expected AggregationTemporality=%d, got %d", aggTemporality, gotTemporality)
	}
	if !gotIsMonotonic {
		t.Errorf("expected IsMonotonic=true, got false")
	}
}

// TestInsertMetadata_NullAggregationFieldsForNonSumSeries verifies a Gauge-shaped
// metadata row stores AggregationTemporality/IsMonotonic as SQL NULL, not a zero value.
func TestInsertMetadata_NullAggregationFieldsForNonSumSeries(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	id := computeMetadataID(canonicalMetadataKey("meta.gauge.only", metricTypeGauge, "1", "scope", "1.0", nil, nil, nil))
	row := MetadataRow{
		MetadataID:   id,
		ScopeName:    "scope",
		ScopeVersion: "1.0",
		ServiceName:  "svc",
		MetricName:   "meta.gauge.only",
		MetricType:   metricTypeGauge,
		MetricUnit:   "1",
		UpdatedAt:    time.Now().UTC(),
		// AggregationTemporality/IsMonotonic left nil - not a Sum series.
	}
	if err := store.InsertMetadata(ctx, []MetadataRow{row}); err != nil {
		t.Fatalf("inserting metadata row: %v", err)
	}

	var (
		gotTemporality sql.NullInt32
		gotMonotonic   sql.NullBool
	)
	err := store.conn.QueryRow(ctx,
		"SELECT AggregationTemporality, IsMonotonic FROM otel_metrics_metadata WHERE MetricName = 'meta.gauge.only'",
	).Scan(&gotTemporality, &gotMonotonic)
	if err != nil {
		t.Fatalf("querying metadata: %v", err)
	}
	if gotTemporality.Valid {
		t.Errorf("expected AggregationTemporality to be NULL for a non-Sum series, got %d", gotTemporality.Int32)
	}
	if gotMonotonic.Valid {
		t.Errorf("expected IsMonotonic to be NULL for a non-Sum series, got %v", gotMonotonic.Bool)
	}
}

// TestInsertGaugeV2 verifies a GaugeV2Row round-trips through otel_metrics_gauge_v2.
func TestInsertGaugeV2(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	now := time.Now().UTC()
	const testID = uint64(424242)
	row := GaugeV2Row{
		MetadataID:    testID,
		StartTimeUnix: now.Add(-time.Minute),
		TimeUnix:      now,
		Value:         77.7,
	}
	if err := store.InsertGaugeV2(ctx, []GaugeV2Row{row}); err != nil {
		t.Fatalf("inserting gauge_v2 row: %v", err)
	}

	var (
		gotID    uint64
		gotValue float64
	)
	err := store.conn.QueryRow(ctx,
		"SELECT MetadataID, Value FROM otel_metrics_gauge_v2 WHERE MetadataID = $1", testID,
	).Scan(&gotID, &gotValue)
	if err != nil {
		t.Fatalf("querying gauge_v2: %v", err)
	}
	if gotID != testID {
		t.Errorf("expected MetadataID=%d, got %d", testID, gotID)
	}
	if gotValue != 77.7 {
		t.Errorf("expected Value=77.7, got %f", gotValue)
	}
}

// TestInsertSumV2 verifies a SumV2Row round-trips through otel_metrics_sum_v2.
func TestInsertSumV2(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	now := time.Now().UTC()
	const testID = uint64(515151)
	row := SumV2Row{
		MetadataID:    testID,
		StartTimeUnix: now.Add(-time.Minute),
		TimeUnix:      now,
		Value:         9001,
	}
	if err := store.InsertSumV2(ctx, []SumV2Row{row}); err != nil {
		t.Fatalf("inserting sum_v2 row: %v", err)
	}

	var (
		gotID    uint64
		gotValue float64
	)
	err := store.conn.QueryRow(ctx,
		"SELECT MetadataID, Value FROM otel_metrics_sum_v2 WHERE MetadataID = $1", testID,
	).Scan(&gotID, &gotValue)
	if err != nil {
		t.Fatalf("querying sum_v2: %v", err)
	}
	if gotID != testID {
		t.Errorf("expected MetadataID=%d, got %d", testID, gotID)
	}
	if gotValue != 9001 {
		t.Errorf("expected Value=9001, got %f", gotValue)
	}
}

// TestMetadataReplacingMergeTree_DedupesOnMerge verifies ReplacingMergeTree
// collapses duplicate rows to the one with the latest UpdatedAt. Dedup identity
// is the full ORDER BY tuple (ServiceName, MetricName, MetadataID), not MetadataID
// alone, so the two rows here share those and vary MetricDescription instead.
func TestMetadataReplacingMergeTree_DedupesOnMerge(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	const testID = uint64(999999)
	older := MetadataRow{
		MetadataID:        testID,
		ServiceName:       "svc",
		MetricName:        "dedup.test",
		MetricDescription: "stale description",
		UpdatedAt:         time.Now().UTC().Add(-time.Hour),
	}
	newer := MetadataRow{
		MetadataID:        testID,
		ServiceName:       "svc",
		MetricName:        "dedup.test",
		MetricDescription: "current description",
		UpdatedAt:         time.Now().UTC(),
	}

	// Insert out of order to make sure dedup is based on UpdatedAt, not
	// insertion order.
	if err := store.InsertMetadata(ctx, []MetadataRow{newer}); err != nil {
		t.Fatalf("inserting newer row: %v", err)
	}
	if err := store.InsertMetadata(ctx, []MetadataRow{older}); err != nil {
		t.Fatalf("inserting older row: %v", err)
	}

	if err := store.conn.Exec(ctx, "OPTIMIZE TABLE otel_metrics_metadata FINAL"); err != nil {
		t.Fatalf("forcing merge: %v", err)
	}

	var count uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_metadata WHERE MetadataID = $1", testID,
	).Scan(&count); err != nil {
		t.Fatalf("counting rows for MetadataID: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected ReplacingMergeTree to collapse to 1 row after OPTIMIZE FINAL, got %d", count)
	}

	var gotDescription string
	if err := store.conn.QueryRow(ctx,
		"SELECT MetricDescription FROM otel_metrics_metadata WHERE MetadataID = $1", testID,
	).Scan(&gotDescription); err != nil {
		t.Fatalf("querying surviving row: %v", err)
	}
	if gotDescription != "current description" {
		t.Errorf("expected the row with the latest UpdatedAt to survive (MetricDescription=%q), got %q", "current description", gotDescription)
	}
}

// dialGRPCTestServer starts a gRPC server wired to store over an in-memory
// bufconn listener, using a series cache with a 10-minute TTL, and returns a
// client dialed to it. Shared by every end-to-end test in this file that
// doesn't care about TTL timing.
func dialGRPCTestServer(t *testing.T, store *ClickHouseMetricsStore) colmetricspb.MetricsServiceClient {
	t.Helper()
	return dialGRPCTestServerWithTTL(t, store, 10*time.Minute)
}

// dialGRPCTestServerWithTTL is dialGRPCTestServer with a caller-supplied
// series cache TTL, for tests that need to observe TTL expiry without
// waiting out the real 10-minute default.
func dialGRPCTestServerWithTTL(t *testing.T, store *ClickHouseMetricsStore, cacheTTL time.Duration) colmetricspb.MetricsServiceClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	colmetricspb.RegisterMetricsServiceServer(grpcServer, newServer("bufconn", store, newSeriesCache(cacheTTL)))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("connecting to grpc server: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return colmetricspb.NewMetricsServiceClient(conn)
}

// TestGRPCToClickHouse verifies a real gRPC Export call writes to both the
// legacy table and the normalized v2 tables, end to end.
func TestGRPCToClickHouse(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	client := dialGRPCTestServer(t, store)

	// Send a gauge metric via gRPC.
	now := uint64(time.Now().UnixNano())
	_, err := client.Export(ctx, &colmetricspb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "e2e-service"}}},
					},
				},
				ScopeMetrics: []*metricspb.ScopeMetrics{
					{
						Scope: &commonpb.InstrumentationScope{Name: "e2e-scope"},
						Metrics: []*metricspb.Metric{
							{
								Name: "e2e.gauge",
								Data: &metricspb.Metric_Gauge{
									Gauge: &metricspb.Gauge{
										DataPoints: []*metricspb.NumberDataPoint{
											{
												TimeUnixNano: now,
												Value:        &metricspb.NumberDataPoint_AsDouble{AsDouble: 99.9},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("exporting metrics via grpc: %v", err)
	}

	// Verify the metric landed in ClickHouse.
	var (
		svcName    string
		metricName string
		value      float64
	)
	err = store.conn.QueryRow(ctx,
		"SELECT ServiceName, MetricName, Value FROM otel_metrics_gauge WHERE MetricName = 'e2e.gauge'",
	).Scan(&svcName, &metricName, &value)
	if err != nil {
		t.Fatalf("querying clickhouse: %v", err)
	}
	if svcName != "e2e-service" {
		t.Errorf("expected ServiceName=e2e-service, got %s", svcName)
	}
	if value != 99.9 {
		t.Errorf("expected Value=99.9, got %f", value)
	}

	// Verify the same call also landed in the normalized v2 tables via the
	// dual-write path in Export, joined back to its metadata by MetadataID.
	wantID := computeMetadataID(canonicalMetadataKey(
		"e2e.gauge", metricTypeGauge, "", "e2e-scope", "",
		map[string]string{"service.name": "e2e-service"}, map[string]string{}, map[string]string{},
	))

	var (
		v2ID    uint64
		v2Value float64
	)
	err = store.conn.QueryRow(ctx,
		"SELECT MetadataID, Value FROM otel_metrics_gauge_v2 WHERE MetadataID = $1", wantID,
	).Scan(&v2ID, &v2Value)
	if err != nil {
		t.Fatalf("querying gauge_v2: %v", err)
	}
	if v2Value != 99.9 {
		t.Errorf("expected v2 Value=99.9, got %f", v2Value)
	}

	var v2ServiceName string
	err = store.conn.QueryRow(ctx,
		"SELECT m.ServiceName FROM otel_metrics_metadata m WHERE m.MetadataID = $1", v2ID,
	).Scan(&v2ServiceName)
	if err != nil {
		t.Fatalf("querying metadata for v2 series: %v", err)
	}
	if v2ServiceName != "e2e-service" {
		t.Errorf("expected v2 ServiceName=e2e-service, got %s", v2ServiceName)
	}
}

// TestGRPCToClickHouse_TwoDifferentSeriesEachWriteMetadata verifies two gRPC
// Export calls for two distinct series (same metric, different attribute
// value) each get their own MetadataID, and otel_metrics_metadata ends up
// with exactly two rows - one per series.
func TestGRPCToClickHouse_TwoDifferentSeriesEachWriteMetadata(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	client := dialGRPCTestServer(t, store)

	if _, err := client.Export(ctx, gaugeExportRequest("e2e-service", "e2e.two-series", map[string]string{"region": "eu"}, 1)); err != nil {
		t.Fatalf("exporting eu series: %v", err)
	}
	if _, err := client.Export(ctx, gaugeExportRequest("e2e-service", "e2e.two-series", map[string]string{"region": "us"}, 2)); err != nil {
		t.Fatalf("exporting us series: %v", err)
	}

	var count uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_metadata WHERE MetricName = 'e2e.two-series'",
	).Scan(&count); err != nil {
		t.Fatalf("counting metadata rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 distinct metadata rows for 2 distinct series, got %d", count)
	}

	// gaugeExportRequest (metrics_service_test.go) always uses scope="scope",
	// scopeVersion="1.0", unit="1" - matching that here to compute the
	// expected IDs independently of the mapper.
	euID := computeMetadataID(canonicalMetadataKey("e2e.two-series", metricTypeGauge, "1", "scope", "1.0",
		map[string]string{"service.name": "e2e-service"}, map[string]string{}, map[string]string{"region": "eu"}))
	usID := computeMetadataID(canonicalMetadataKey("e2e.two-series", metricTypeGauge, "1", "scope", "1.0",
		map[string]string{"service.name": "e2e-service"}, map[string]string{}, map[string]string{"region": "us"}))
	if euID == usID {
		t.Fatalf("test setup invalid: eu and us series produced the same MetadataID")
	}

	for _, id := range []uint64{euID, usID} {
		var exists uint64
		if err := store.conn.QueryRow(ctx,
			"SELECT count() FROM otel_metrics_metadata WHERE MetadataID = $1", id,
		).Scan(&exists); err != nil {
			t.Fatalf("querying MetadataID %d: %v", id, err)
		}
		if exists != 1 {
			t.Errorf("expected exactly 1 metadata row for MetadataID %d, got %d", id, exists)
		}
	}
}

// TestGRPCToClickHouse_SameSeriesAcrossRequestsWritesMetadataOnce verifies
// two separate gRPC Export calls for the same series (identical metadata,
// different data point values) hash to the same MetadataID and touch
// otel_metrics_metadata only once - the series cache skips the second
// metadata write - while both data points still land in the fact table.
func TestGRPCToClickHouse_SameSeriesAcrossRequestsWritesMetadataOnce(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	client := dialGRPCTestServer(t, store)

	if _, err := client.Export(ctx, gaugeExportRequest("e2e-service", "e2e.same-series", map[string]string{"region": "eu"}, 10)); err != nil {
		t.Fatalf("exporting first point: %v", err)
	}
	if _, err := client.Export(ctx, gaugeExportRequest("e2e-service", "e2e.same-series", map[string]string{"region": "eu"}, 20)); err != nil {
		t.Fatalf("exporting second point: %v", err)
	}

	wantID := computeMetadataID(canonicalMetadataKey("e2e.same-series", metricTypeGauge, "1", "scope", "1.0",
		map[string]string{"service.name": "e2e-service"}, map[string]string{}, map[string]string{"region": "eu"}))

	var metadataCount uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_metadata WHERE MetricName = 'e2e.same-series'",
	).Scan(&metadataCount); err != nil {
		t.Fatalf("counting metadata rows: %v", err)
	}
	if metadataCount != 1 {
		t.Fatalf("expected otel_metrics_metadata to be touched exactly once across both requests, got %d rows", metadataCount)
	}

	var gotID uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT MetadataID FROM otel_metrics_metadata WHERE MetricName = 'e2e.same-series'",
	).Scan(&gotID); err != nil {
		t.Fatalf("querying metadata row: %v", err)
	}
	if gotID != wantID {
		t.Errorf("expected MetadataID=%d, got %d", wantID, gotID)
	}

	// The cache only skips redundant metadata writes, not data point writes -
	// both points should still be present in the fact table.
	var dataPointCount uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_gauge_v2 WHERE MetadataID = $1", wantID,
	).Scan(&dataPointCount); err != nil {
		t.Fatalf("counting gauge_v2 rows: %v", err)
	}
	if dataPointCount != 2 {
		t.Errorf("expected 2 data point rows for the shared MetadataID, got %d", dataPointCount)
	}
}

// TestGRPCToClickHouse_DescriptionChangeAfterTTLCreatesSecondRowUntilMerged
// verifies that MetricDescription isn't part of the MetadataID hash, so a
// changed description for the same series only reaches ClickHouse once the
// series cache TTL has expired - and even then, both rows physically exist
// until a merge actually happens (forced here via OPTIMIZE FINAL), since
// ReplacingMergeTree doesn't dedupe synchronously at insert time. Uses a
// short TTL so the test doesn't have to wait out a real 10-minute window.
func TestGRPCToClickHouse_DescriptionChangeAfterTTLCreatesSecondRowUntilMerged(t *testing.T) {
	store, cleanup := setupClickHouse(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("creating tables: %v", err)
	}

	const ttl = 50 * time.Millisecond
	client := dialGRPCTestServerWithTTL(t, store, ttl)

	buildRequest := func(description string, value float64) *colmetricspb.ExportMetricsServiceRequest {
		now := uint64(time.Now().UnixNano())
		return &colmetricspb.ExportMetricsServiceRequest{
			ResourceMetrics: []*metricspb.ResourceMetrics{
				newResourceMetrics("e2e-service", nil, "scope", "1.0", nil,
					newGaugeMetric("e2e.description-change", "1", description,
						newDataPoint(map[string]string{"region": "eu"}, now-uint64(time.Minute), now, value)),
				),
			},
		}
	}

	if _, err := client.Export(ctx, buildRequest("original description", 1)); err != nil {
		t.Fatalf("exporting first request: %v", err)
	}

	// Wait past the cache TTL so the next request for this series isn't suppressed.
	time.Sleep(ttl + 20*time.Millisecond)

	if _, err := client.Export(ctx, buildRequest("changed description", 2)); err != nil {
		t.Fatalf("exporting second request: %v", err)
	}

	wantID := computeMetadataID(canonicalMetadataKey("e2e.description-change", metricTypeGauge, "1", "scope", "1.0",
		map[string]string{"service.name": "e2e-service"}, map[string]string{}, map[string]string{"region": "eu"}))

	// Immediately after the second write, both rows physically exist -
	// ReplacingMergeTree does not dedupe synchronously at insert time.
	var rowCount uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_metadata WHERE MetadataID = $1", wantID,
	).Scan(&rowCount); err != nil {
		t.Fatalf("counting metadata rows: %v", err)
	}
	if rowCount != 2 {
		t.Fatalf("expected 2 physical rows for the same MetadataID before any merge, got %d", rowCount)
	}

	if err := store.conn.Exec(ctx, "OPTIMIZE TABLE otel_metrics_metadata FINAL"); err != nil {
		t.Fatalf("forcing merge: %v", err)
	}

	var mergedCount uint64
	if err := store.conn.QueryRow(ctx,
		"SELECT count() FROM otel_metrics_metadata WHERE MetadataID = $1", wantID,
	).Scan(&mergedCount); err != nil {
		t.Fatalf("counting merged rows: %v", err)
	}
	if mergedCount != 1 {
		t.Fatalf("expected ReplacingMergeTree to collapse to 1 row after OPTIMIZE FINAL, got %d", mergedCount)
	}

	var gotDescription string
	if err := store.conn.QueryRow(ctx,
		"SELECT MetricDescription FROM otel_metrics_metadata WHERE MetadataID = $1", wantID,
	).Scan(&gotDescription); err != nil {
		t.Fatalf("querying surviving row: %v", err)
	}
	if gotDescription != "changed description" {
		t.Errorf("expected the surviving row to have the latest description, got %q", gotDescription)
	}
}
