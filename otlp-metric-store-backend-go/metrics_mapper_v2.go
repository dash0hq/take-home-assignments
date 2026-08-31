package main

import (
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// Metric type strings, used both as a hash input (canonicalMetadataKey) and
// as the stored MetricType value - so a Gauge and Sum sharing an otherwise
// identical name/unit/scope/attributes get distinct MetadataIDs instead of
// silently colliding.
const (
	metricTypeGauge = "gauge"
	metricTypeSum   = "sum"
)

// newMetadataRow builds the metadata row shared by every metric type,
// reusing the same serviceName/kvToMap helpers already used by the legacy
// mapper functions in metrics_mapper.go. aggTemporality/isMonotonic are only
// meaningful for Sum series - callers pass nil for every other metric type,
// which stores as NULL (see the Nullable columns on otel_metrics_metadata).
func newMetadataRow(
	id uint64,
	svcName string,
	metricType string,
	metric *metricspb.Metric,
	resAttrs map[string]string,
	resSchemaUrl string,
	scope *commonpb.InstrumentationScope,
	scopeAttrs map[string]string,
	scopeSchemaUrl string,
	dpAttrs map[string]string,
	aggTemporality *int32,
	isMonotonic *bool,
	now time.Time,
) MetadataRow {
	return MetadataRow{
		MetadataID:             id,
		ResourceAttributes:     resAttrs,
		ResourceSchemaUrl:      resSchemaUrl,
		ScopeName:              scope.GetName(),
		ScopeVersion:           scope.GetVersion(),
		ScopeAttributes:        scopeAttrs,
		ScopeDroppedAttrCount:  scope.GetDroppedAttributesCount(),
		ScopeSchemaUrl:         scopeSchemaUrl,
		ServiceName:            svcName,
		MetricName:             metric.GetName(),
		MetricType:             metricType,
		MetricDescription:      metric.GetDescription(),
		MetricUnit:             metric.GetUnit(),
		AggregationTemporality: aggTemporality,
		IsMonotonic:            isMonotonic,
		Attributes:             dpAttrs,
		UpdatedAt:              now,
	}
}

// MapMetricsV2 converts an ExportMetricsServiceRequest's ResourceMetrics into
// the normalized schema: data-point rows for otel_metrics_gauge_v2 and
// otel_metrics_sum_v2, plus metadata rows for any series the cache hasn't
// confirmed recently (see series_cache.go). This runs alongside — not
// instead of — the legacy MapGaugeRows/MapSumRows during the migration
// bridge period; the two mappers are independent and neither depends on
// the other's output.
func MapMetricsV2(
	resourceMetrics []*metricspb.ResourceMetrics,
	cache *seriesCache,
	now time.Time,
) (metadata []MetadataRow, gauges []GaugeV2Row, sums []SumV2Row) {
	for _, rm := range resourceMetrics {
		svcName := serviceName(rm.GetResource())
		resAttrs := kvToMap(rm.GetResource().GetAttributes())
		resSchemaUrl := rm.GetSchemaUrl()

		for _, sm := range rm.GetScopeMetrics() {
			scope := sm.GetScope()
			scopeAttrs := kvToMap(scope.GetAttributes())
			scopeSchemaUrl := sm.GetSchemaUrl()

			for _, metric := range sm.GetMetrics() {
				switch data := metric.GetData().(type) {

				case *metricspb.Metric_Gauge:
					for _, dp := range data.Gauge.GetDataPoints() {
						dpAttrs := kvToMap(dp.GetAttributes())
						key := canonicalMetadataKey(
							metric.GetName(), metricTypeGauge, metric.GetUnit(),
							scope.GetName(), scope.GetVersion(),
							resAttrs, scopeAttrs, dpAttrs,
						)
						id := computeMetadataID(key)

						gauges = append(gauges, GaugeV2Row{
							MetadataID:    id,
							StartTimeUnix: nanosToTime(dp.GetStartTimeUnixNano()),
							TimeUnix:      nanosToTime(dp.GetTimeUnixNano()),
							Value:         numberDataPointValue(dp),
							Flags:         dp.GetFlags(),
						})

						if cache.shouldWrite(id, now) {
							metadata = append(metadata, newMetadataRow(
								id, svcName, metricTypeGauge, metric, resAttrs, resSchemaUrl,
								scope, scopeAttrs, scopeSchemaUrl, dpAttrs,
								nil, nil, now,
							))
						}
					}

				case *metricspb.Metric_Sum:
					aggTemporality := int32(data.Sum.GetAggregationTemporality())
					isMonotonic := data.Sum.GetIsMonotonic()

					for _, dp := range data.Sum.GetDataPoints() {
						dpAttrs := kvToMap(dp.GetAttributes())
						key := canonicalMetadataKey(
							metric.GetName(), metricTypeSum, metric.GetUnit(),
							scope.GetName(), scope.GetVersion(),
							resAttrs, scopeAttrs, dpAttrs,
						)
						id := computeMetadataID(key)

						sums = append(sums, SumV2Row{
							MetadataID:    id,
							StartTimeUnix: nanosToTime(dp.GetStartTimeUnixNano()),
							TimeUnix:      nanosToTime(dp.GetTimeUnixNano()),
							Value:         numberDataPointValue(dp),
							Flags:         dp.GetFlags(),
						})

						if cache.shouldWrite(id, now) {
							metadata = append(metadata, newMetadataRow(
								id, svcName, metricTypeSum, metric, resAttrs, resSchemaUrl,
								scope, scopeAttrs, scopeSchemaUrl, dpAttrs,
								&aggTemporality, &isMonotonic, now,
							))
						}
					}
				}
			}
		}
	}
	return metadata, gauges, sums
}
