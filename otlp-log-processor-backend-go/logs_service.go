package main

import (
	"context"
	"log/slog"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dash0LogsServiceServer struct {
	addr      string
	processor *LogProcessor

	collogspb.UnimplementedLogsServiceServer
}

func newServer(addr string, processor *LogProcessor) collogspb.LogsServiceServer {
	s := &dash0LogsServiceServer{
		addr:      addr,
		processor: processor,
	}
	return s
}

func (l *dash0LogsServiceServer) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	slog.DebugContext(ctx, "Received ExportLogsServiceRequest")
	logsReceivedCounter.Add(ctx, 1)

	if err := l.processor.ProcessLogs(ctx, request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &collogspb.ExportLogsServiceResponse{}, nil
}
