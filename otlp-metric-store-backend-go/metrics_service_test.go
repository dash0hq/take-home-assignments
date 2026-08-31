package main

import (
	"context"
	"sync"
	"testing"
	"time"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// fakeMetricsStore records call volume instead of touching ClickHouse, so
// Export's wiring (legacy path + v2 dual-write path + metadata cache) can be
// verified without a real database.
type fakeMetricsStore struct {
	mu sync.Mutex

	gaugeInserts    int
	sumInserts      int
	metadataInserts int
	gaugeV2Inserts  int
	sumV2Inserts    int
}

func (f *fakeMetricsStore) CreateTables(context.Context) error { return nil }

func (f *fakeMetricsStore) InsertGauge(_ context.Context, rows []GaugeRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gaugeInserts += len(rows)
	return nil
}

func (f *fakeMetricsStore) InsertSum(_ context.Context, rows []SumRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sumInserts += len(rows)
	return nil
}

func (f *fakeMetricsStore) InsertMetadata(_ context.Context, rows []MetadataRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.metadataInserts += len(rows)
	return nil
}

func (f *fakeMetricsStore) InsertGaugeV2(_ context.Context, rows []GaugeV2Row) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gaugeV2Inserts += len(rows)
	return nil
}

func (f *fakeMetricsStore) InsertSumV2(_ context.Context, rows []SumV2Row) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sumV2Inserts += len(rows)
	return nil
}

func (f *fakeMetricsStore) Close() error { return nil }

func gaugeExportRequest(serviceName, metricName string, dpAttrs map[string]string, value float64) *colmetricspb.ExportMetricsServiceRequest {
	now := uint64(time.Now().UnixNano())
	return &colmetricspb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			newResourceMetrics(serviceName, nil, "scope", "1.0", nil,
				newGaugeMetric(metricName, "1", "", newDataPoint(dpAttrs, now-uint64(time.Minute), now, value)),
			),
		},
	}
}

// TestExport_WritesBothLegacyAndV2Paths verifies a single Export call writes
// to both the legacy table and the normalized v2 tables (data + metadata).
func TestExport_WritesBothLegacyAndV2Paths(t *testing.T) {
	store := &fakeMetricsStore{}
	server := newServer("test", store, newSeriesCache(10*time.Minute))
	ctx := context.Background()

	req := gaugeExportRequest("svc", "requests", map[string]string{"region": "eu"}, 1)
	if _, err := server.Export(ctx, req); err != nil {
		t.Fatalf("Export: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.gaugeInserts != 1 {
		t.Errorf("legacy InsertGauge calls = %d, want 1", store.gaugeInserts)
	}
	if store.gaugeV2Inserts != 1 {
		t.Errorf("InsertGaugeV2 calls = %d, want 1", store.gaugeV2Inserts)
	}
	if store.metadataInserts != 1 {
		t.Errorf("InsertMetadata calls = %d, want 1", store.metadataInserts)
	}
	if store.sumInserts != 0 || store.sumV2Inserts != 0 {
		t.Errorf("expected no sum inserts for a gauge-only request, got sum=%d sumV2=%d", store.sumInserts, store.sumV2Inserts)
	}
}

// TestExport_MetadataCacheAvoidsDuplicateWriteAcrossCalls verifies repeat
// Exports for the same series keep writing fact-table rows every time, but
// only write metadata once, end-to-end through the real Export method.
func TestExport_MetadataCacheAvoidsDuplicateWriteAcrossCalls(t *testing.T) {
	store := &fakeMetricsStore{}
	server := newServer("test", store, newSeriesCache(10*time.Minute))
	ctx := context.Background()

	req := gaugeExportRequest("svc", "requests", map[string]string{"region": "eu"}, 1)
	if _, err := server.Export(ctx, req); err != nil {
		t.Fatalf("first Export: %v", err)
	}
	if _, err := server.Export(ctx, req); err != nil {
		t.Fatalf("second Export: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	// Legacy path has no cache - it writes every time.
	if store.gaugeInserts != 2 {
		t.Errorf("legacy InsertGauge calls = %d, want 2", store.gaugeInserts)
	}
	if store.gaugeV2Inserts != 2 {
		t.Errorf("InsertGaugeV2 calls = %d, want 2 (fact table always writes)", store.gaugeV2Inserts)
	}
	// v2 metadata path is cached - only the first call should write it.
	if store.metadataInserts != 1 {
		t.Errorf("InsertMetadata calls = %d, want 1 across both exports", store.metadataInserts)
	}
}

// TestExport_DifferentSeriesEachWriteOwnMetadata verifies two Exports for
// distinct series (different attribute values) each write their own metadata.
func TestExport_DifferentSeriesEachWriteOwnMetadata(t *testing.T) {
	store := &fakeMetricsStore{}
	server := newServer("test", store, newSeriesCache(10*time.Minute))
	ctx := context.Background()

	if _, err := server.Export(ctx, gaugeExportRequest("svc", "requests", map[string]string{"region": "eu"}, 1)); err != nil {
		t.Fatalf("Export eu: %v", err)
	}
	if _, err := server.Export(ctx, gaugeExportRequest("svc", "requests", map[string]string{"region": "us"}, 1)); err != nil {
		t.Fatalf("Export us: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.metadataInserts != 2 {
		t.Errorf("InsertMetadata calls = %d, want 2 for 2 distinct series", store.metadataInserts)
	}
}

// TestExport_NilStoreIsNoop verifies Export succeeds without touching any
// store when the server was constructed with a nil store.
func TestExport_NilStoreIsNoop(t *testing.T) {
	server := newServer("test", nil, newSeriesCache(10*time.Minute))
	ctx := context.Background()

	resp, err := server.Export(ctx, gaugeExportRequest("svc", "requests", nil, 1))
	if err != nil {
		t.Fatalf("expected no error with a nil store, got %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a non-nil response with a nil store")
	}
}
