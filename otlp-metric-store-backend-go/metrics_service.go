package main

import (
	"context"
	"log/slog"
	"time"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
)

type dash0MetricsServiceServer struct {
	addr  string
	store MetricsStore
	cache *seriesCache

	colmetricspb.UnimplementedMetricsServiceServer
}

func newServer(addr string, store MetricsStore, cache *seriesCache) colmetricspb.MetricsServiceServer {
	return &dash0MetricsServiceServer{addr: addr, store: store, cache: cache}
}

func (m *dash0MetricsServiceServer) Export(ctx context.Context, request *colmetricspb.ExportMetricsServiceRequest) (*colmetricspb.ExportMetricsServiceResponse, error) {
	rm := request.GetResourceMetrics()
	slog.InfoContext(ctx, "received ExportMetricsServiceRequest", slog.Int("resourceMetrics", len(rm)))
	metricsReceivedCounter.Add(ctx, 1)

	if m.store == nil {
		return &colmetricspb.ExportMetricsServiceResponse{}, nil
	}

	// Legacy path - writes the existing, unmodified tables.
	gaugeRows := MapGaugeRows(rm)
	if len(gaugeRows) > 0 {
		if err := m.store.InsertGauge(ctx, gaugeRows); err != nil {
			slog.ErrorContext(ctx, "inserting legacy gauge rows failed", slog.Int("rows", len(gaugeRows)), slog.Any("error", err))
			return nil, err
		}
	}
	sumRows := MapSumRows(rm)
	if len(sumRows) > 0 {
		if err := m.store.InsertSum(ctx, sumRows); err != nil {
			slog.ErrorContext(ctx, "inserting legacy sum rows failed", slog.Int("rows", len(sumRows)), slog.Any("error", err))
			return nil, err
		}
	}

	// Normalized v2 path - temporary migration bridge, written
	// alongside the legacy path above until consumers migrate off the
	// old tables. Metadata is inserted before the fact-table rows so a
	// reader querying immediately after this call sees metadata
	// already resolvable for any MetadataID it might encounter.
	now := time.Now()
	metadataRows, gaugeV2Rows, sumV2Rows := MapMetricsV2(rm, m.cache, now)

	if err := m.store.InsertMetadata(ctx, metadataRows); err != nil {
		slog.ErrorContext(ctx, "inserting metadata rows failed", slog.Int("rows", len(metadataRows)), slog.Any("error", err))
		return nil, err
	}
	if err := m.store.InsertGaugeV2(ctx, gaugeV2Rows); err != nil {
		slog.ErrorContext(ctx, "inserting v2 gauge rows failed", slog.Int("rows", len(gaugeV2Rows)), slog.Any("error", err))
		return nil, err
	}
	if err := m.store.InsertSumV2(ctx, sumV2Rows); err != nil {
		slog.ErrorContext(ctx, "inserting v2 sum rows failed", slog.Int("rows", len(sumV2Rows)), slog.Any("error", err))
		return nil, err
	}

	slog.InfoContext(ctx, "export completed",
		slog.Int("gaugeRows", len(gaugeRows)),
		slog.Int("sumRows", len(sumRows)),
		slog.Int("metadataRows", len(metadataRows)),
		slog.Int("gaugeV2Rows", len(gaugeV2Rows)),
		slog.Int("sumV2Rows", len(sumV2Rows)),
	)

	return &colmetricspb.ExportMetricsServiceResponse{}, nil
}
