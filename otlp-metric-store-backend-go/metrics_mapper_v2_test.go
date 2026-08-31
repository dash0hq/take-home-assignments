package main

import (
	"maps"
	"testing"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

// -----------------------------------------------------------------------------
// Shared OTLP construction helpers, also used by metrics_service_test.go.
// Composable on purpose: each test builds exactly the shape it needs out of
// these small pieces instead of a large parameter list or config struct.
// -----------------------------------------------------------------------------

func kv(key, value string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: value}}}
}

func kvMap(attrs map[string]string) []*commonpb.KeyValue {
	out := make([]*commonpb.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		out = append(out, kv(k, v))
	}
	return out
}

func newDataPoint(attrs map[string]string, startNano, timeNano uint64, value float64) *metricspb.NumberDataPoint {
	return &metricspb.NumberDataPoint{
		Attributes:        kvMap(attrs),
		StartTimeUnixNano: startNano,
		TimeUnixNano:      timeNano,
		Value:             &metricspb.NumberDataPoint_AsDouble{AsDouble: value},
	}
}

func newGaugeMetric(name, unit, description string, dataPoints ...*metricspb.NumberDataPoint) *metricspb.Metric {
	return &metricspb.Metric{
		Name:        name,
		Unit:        unit,
		Description: description,
		Data:        &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: dataPoints}},
	}
}

func newSumMetric(name, unit, description string, temporality metricspb.AggregationTemporality, isMonotonic bool, dataPoints ...*metricspb.NumberDataPoint) *metricspb.Metric {
	return &metricspb.Metric{
		Name:        name,
		Unit:        unit,
		Description: description,
		Data: &metricspb.Metric_Sum{Sum: &metricspb.Sum{
			AggregationTemporality: temporality,
			IsMonotonic:            isMonotonic,
			DataPoints:             dataPoints,
		}},
	}
}

func newResourceMetrics(serviceName string, resAttrs map[string]string, scopeName, scopeVersion string, scopeAttrs map[string]string, metrics ...*metricspb.Metric) *metricspb.ResourceMetrics {
	allResAttrs := map[string]string{"service.name": serviceName}
	maps.Copy(allResAttrs, resAttrs)
	return &metricspb.ResourceMetrics{
		Resource: &resourcepb.Resource{Attributes: kvMap(allResAttrs)},
		ScopeMetrics: []*metricspb.ScopeMetrics{
			{
				Scope:   &commonpb.InstrumentationScope{Name: scopeName, Version: scopeVersion, Attributes: kvMap(scopeAttrs)},
				Metrics: metrics,
			},
		},
	}
}

// TestMapMetricsV2_GaugeMapping verifies a Gauge data point maps to a
// GaugeV2Row plus a matching MetadataRow, both keyed by the same MetadataID.
func TestMapMetricsV2_GaugeMapping(t *testing.T) {
	now := time.Now()
	nowNano := uint64(now.UnixNano())
	startNano := nowNano - uint64(time.Minute)

	resAttrs := map[string]string{"host.name": "h1"}
	scopeAttrs := map[string]string{"lib": "otel-go"}
	dpAttrs := map[string]string{"cpu": "0"}

	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", resAttrs, "scope", "1.0", scopeAttrs,
			newGaugeMetric("cpu.util", "%", "CPU utilization", newDataPoint(dpAttrs, startNano, nowNano, 42.5)),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, sums := MapMetricsV2(rm, cache, now)

	if len(sums) != 0 {
		t.Fatalf("expected 0 sum rows, got %d", len(sums))
	}
	if len(gauges) != 1 {
		t.Fatalf("expected 1 gauge row, got %d", len(gauges))
	}
	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata row, got %d", len(metadata))
	}

	wantID := computeMetadataID(canonicalMetadataKey(
		"cpu.util", metricTypeGauge, "%", "scope", "1.0",
		map[string]string{"service.name": "svc", "host.name": "h1"},
		scopeAttrs, dpAttrs,
	))

	g := gauges[0]
	if g.MetadataID != wantID {
		t.Errorf("GaugeV2Row.MetadataID = %d, want %d", g.MetadataID, wantID)
	}
	if g.Value != 42.5 {
		t.Errorf("GaugeV2Row.Value = %f, want 42.5", g.Value)
	}
	if !g.TimeUnix.Equal(nanosToTime(nowNano)) || !g.StartTimeUnix.Equal(nanosToTime(startNano)) {
		t.Errorf("GaugeV2Row timestamps not mapped correctly")
	}

	m := metadata[0]
	if m.MetadataID != wantID {
		t.Errorf("MetadataRow.MetadataID = %d, want %d", m.MetadataID, wantID)
	}
	if m.ServiceName != "svc" || m.MetricName != "cpu.util" || m.MetricUnit != "%" || m.MetricDescription != "CPU utilization" {
		t.Errorf("MetadataRow fields incorrect: %+v", m)
	}
	if m.Attributes["cpu"] != "0" {
		t.Errorf("MetadataRow.Attributes = %v, want cpu=0", m.Attributes)
	}
}

// TestMapMetricsV2_SumMapping verifies a Sum data point maps to a SumV2Row,
// with AggregationTemporality/IsMonotonic landing on the metadata row instead.
func TestMapMetricsV2_SumMapping(t *testing.T) {
	now := time.Now()
	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", nil, "scope", "1.0", nil,
			newSumMetric("http.requests", "{request}", "Total requests",
				metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE, true,
				newDataPoint(map[string]string{"method": "GET"}, 0, uint64(now.UnixNano()), 1234)),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, sums := MapMetricsV2(rm, cache, now)

	if len(gauges) != 0 {
		t.Fatalf("expected 0 gauge rows, got %d", len(gauges))
	}
	if len(sums) != 1 {
		t.Fatalf("expected 1 sum row, got %d", len(sums))
	}
	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata row, got %d", len(metadata))
	}

	s := sums[0]
	if s.Value != 1234 {
		t.Errorf("SumV2Row.Value = %f, want 1234", s.Value)
	}

	// AggregationTemporality/IsMonotonic are series-level, so they live on
	// the metadata row, not on the per-point SumV2Row.
	m := metadata[0]
	wantTemporality := int32(metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
	if m.AggregationTemporality == nil || *m.AggregationTemporality != wantTemporality {
		t.Errorf("MetadataRow.AggregationTemporality = %v, want %d", m.AggregationTemporality, wantTemporality)
	}
	if m.IsMonotonic == nil || !*m.IsMonotonic {
		t.Errorf("MetadataRow.IsMonotonic = %v, want true", m.IsMonotonic)
	}
}

// TestMapMetricsV2_GaugeMetadataHasNilAggregationFields verifies a Gauge
// series' metadata leaves AggregationTemporality/IsMonotonic nil (NULL) -
// they're only meaningful for Sum series.
func TestMapMetricsV2_GaugeMetadataHasNilAggregationFields(t *testing.T) {
	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", nil, "scope", "1.0", nil,
			newGaugeMetric("cpu.util", "%", "", newDataPoint(nil, 0, 1, 1)),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, _, _ := MapMetricsV2(rm, cache, time.Now())

	if len(metadata) != 1 {
		t.Fatalf("expected 1 metadata row, got %d", len(metadata))
	}
	m := metadata[0]
	if m.AggregationTemporality != nil {
		t.Errorf("expected Gauge metadata AggregationTemporality to be nil, got %v", *m.AggregationTemporality)
	}
	if m.IsMonotonic != nil {
		t.Errorf("expected Gauge metadata IsMonotonic to be nil, got %v", *m.IsMonotonic)
	}
}

// TestMapMetricsV2_DedupesMetadataWithinBatchViaCache verifies two data
// points for the same series in one batch produce only one metadata row.
func TestMapMetricsV2_DedupesMetadataWithinBatchViaCache(t *testing.T) {
	now := time.Now()
	dpAttrs := map[string]string{"region": "eu"}
	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", nil, "scope", "1.0", nil,
			newGaugeMetric("requests", "1", "",
				newDataPoint(dpAttrs, 0, 1, 1),
				newDataPoint(dpAttrs, 0, 2, 2), // same series, second point in the same batch
			),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, _ := MapMetricsV2(rm, cache, now)

	if len(gauges) != 2 {
		t.Fatalf("expected 2 gauge rows, got %d", len(gauges))
	}
	if len(metadata) != 1 {
		t.Fatalf("expected metadata deduped to 1 row within the batch, got %d", len(metadata))
	}
	if gauges[0].MetadataID != gauges[1].MetadataID {
		t.Errorf("expected both gauge rows to reference the same MetadataID")
	}
}

// TestMapMetricsV2_SkipsMetadataWhenCacheAlreadyKnowsSeries verifies a series
// the cache already confirmed recently gets its data point row, but no
// redundant metadata row.
func TestMapMetricsV2_SkipsMetadataWhenCacheAlreadyKnowsSeries(t *testing.T) {
	now := time.Now()
	resAttrs := map[string]string{}
	scopeAttrs := map[string]string{}
	dpAttrs := map[string]string{"region": "eu"}

	id := computeMetadataID(canonicalMetadataKey(
		"requests", metricTypeGauge, "1", "scope", "1.0",
		map[string]string{"service.name": "svc"}, scopeAttrs, dpAttrs,
	))

	cache := newSeriesCache(10 * time.Minute)
	cache.shouldWrite(id, now.Add(-time.Second)) // pre-seed: already confirmed just now

	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", resAttrs, "scope", "1.0", scopeAttrs,
			newGaugeMetric("requests", "1", "", newDataPoint(dpAttrs, 0, 1, 1)),
		),
	}

	metadata, gauges, _ := MapMetricsV2(rm, cache, now)
	if len(gauges) != 1 {
		t.Fatalf("expected the data point row regardless of cache state, got %d gauge rows", len(gauges))
	}
	if len(metadata) != 0 {
		t.Errorf("expected no metadata row for an already-cached series, got %d", len(metadata))
	}
}

// TestMapMetricsV2_DifferentSeriesEachGetOwnMetadata verifies two data points
// differing only in an attribute value get distinct MetadataIDs and metadata rows.
func TestMapMetricsV2_DifferentSeriesEachGetOwnMetadata(t *testing.T) {
	now := time.Now()
	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", nil, "scope", "1.0", nil,
			newGaugeMetric("requests", "1", "", newDataPoint(map[string]string{"region": "eu"}, 0, 1, 1)),
			newGaugeMetric("requests", "1", "", newDataPoint(map[string]string{"region": "us"}, 0, 1, 2)),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, _ := MapMetricsV2(rm, cache, now)

	if len(gauges) != 2 {
		t.Fatalf("expected 2 gauge rows, got %d", len(gauges))
	}
	if len(metadata) != 2 {
		t.Fatalf("expected 2 distinct metadata rows for 2 distinct series, got %d", len(metadata))
	}
	if gauges[0].MetadataID == gauges[1].MetadataID {
		t.Errorf("expected different attribute values to produce different MetadataIDs")
	}
}

// TestMapMetricsV2_IgnoresOtherMetricTypes verifies a metric type MapMetricsV2
// doesn't handle (e.g. Histogram) is skipped without panicking.
func TestMapMetricsV2_IgnoresOtherMetricTypes(t *testing.T) {
	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", nil, "scope", "1.0", nil,
			&metricspb.Metric{
				Name: "some.histogram",
				Data: &metricspb.Metric_Histogram{Histogram: &metricspb.Histogram{}},
			},
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, sums := MapMetricsV2(rm, cache, time.Now())

	if len(metadata) != 0 || len(gauges) != 0 || len(sums) != 0 {
		t.Errorf("expected MapMetricsV2 to ignore non-Gauge/Sum metric types without panicking, got metadata=%d gauges=%d sums=%d",
			len(metadata), len(gauges), len(sums))
	}
}

// TestMapMetricsV2_GaugeAndSumWithSameNameGetDifferentMetadataIDs verifies a
// Gauge and a Sum sharing an otherwise identical name/unit/scope/attributes
// no longer collide on the same MetadataID, now that MetricType is part of
// the hash - regression coverage for the bug this fixed.
func TestMapMetricsV2_GaugeAndSumWithSameNameGetDifferentMetadataIDs(t *testing.T) {
	resAttrs := map[string]string{"host.name": "h1"}
	dpAttrs := map[string]string{"unit_test": "true"}

	rm := []*metricspb.ResourceMetrics{
		newResourceMetrics("svc", resAttrs, "scope", "1.0", nil,
			newGaugeMetric("ambiguous.metric", "1", "reported as a gauge", newDataPoint(dpAttrs, 0, 1, 1)),
			newSumMetric("ambiguous.metric", "1", "reported as a sum",
				metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE, true,
				newDataPoint(dpAttrs, 0, 1, 2)),
		),
	}

	cache := newSeriesCache(10 * time.Minute)
	metadata, gauges, sums := MapMetricsV2(rm, cache, time.Now())

	if len(gauges) != 1 || len(sums) != 1 {
		t.Fatalf("expected 1 gauge row and 1 sum row, got %d/%d", len(gauges), len(sums))
	}
	if gauges[0].MetadataID == sums[0].MetadataID {
		t.Errorf("expected Gauge and Sum to get different MetadataIDs, both got %d", gauges[0].MetadataID)
	}
	// Both get their own metadata row now, instead of the Sum's write being
	// suppressed by the cache as a false "repeat" of the Gauge's ID.
	if len(metadata) != 2 {
		t.Errorf("expected 2 distinct metadata rows (one per type), got %d", len(metadata))
	}
}
