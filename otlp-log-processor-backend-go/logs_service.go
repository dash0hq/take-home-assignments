package main

import (
	"context"
	"log/slog"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

type dash0LogsServiceServer struct {
	addr string

	collogspb.UnimplementedLogsServiceServer
}

func newServer(addr string) collogspb.LogsServiceServer {
	s := &dash0LogsServiceServer{addr: addr}
	return s
}

func (l *dash0LogsServiceServer) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	slog.DebugContext(ctx, "Received ExportLogsServiceRequest")
	logsReceivedCounter.Add(ctx, 1)

	// Do something with the logs

	return &collogspb.ExportLogsServiceResponse{}, nil
}
