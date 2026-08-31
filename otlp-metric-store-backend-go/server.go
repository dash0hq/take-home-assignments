package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	listenAddr            = flag.String("listenAddr", "localhost:4317", "The listen address")
	maxReceiveMessageSize = flag.Int("maxReceiveMessageSize", 16777216, "The max message size in bytes the server can receive")
)

const name = "dash0.com/otlp-metrics-processor-backend"

var (
	meter                  = otel.Meter(name)
	logger                 = otelslog.NewLogger(name)
	metricsReceivedCounter metric.Int64Counter
)

func init() {
	var err error
	metricsReceivedCounter, err = meter.Int64Counter("com.dash0.homeexercise.metrics.received",
		metric.WithDescription("The number of metrics received by otlp-metrics-processor-backend"),
		metric.WithUnit("{metric}"))
	if err != nil {
		panic(err)
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	slog.SetDefault(logger)
	logger.Info("starting application")

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(context.Background())
	if err != nil {
		slog.Error("setting up OpenTelemetry SDK failed", slog.Any("error", err))
		return
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		slog.Info("shutting down")
		if shutdownErr := otelShutdown(context.Background()); shutdownErr != nil {
			slog.Error("OpenTelemetry shutdown failed", slog.Any("error", shutdownErr))
			err = errors.Join(err, shutdownErr)
		}
	}()

	flag.Parse()

	slog.Info("starting listener", slog.String("listenAddr", *listenAddr))
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		slog.Error("starting listener failed", slog.String("listenAddr", *listenAddr), slog.Any("error", err))
		return err
	}
	slog.Info("listener started", slog.String("listenAddr", *listenAddr))

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.MaxRecvMsgSize(*maxReceiveMessageSize),
		grpc.Creds(insecure.NewCredentials()),
	)

	// TODO: wire up a real ClickHouseMetricsStore here instead of nil - until
	// then, every received metric is silently dropped (see README's Known
	// Limitations). This warning is the one signal an operator currently has
	// that nothing is being persisted.
	slog.Warn("no ClickHouse store configured - received metrics will not be persisted")
	colmetricspb.RegisterMetricsServiceServer(grpcServer, newServer(*listenAddr, nil, newSeriesCache(10*time.Minute)))

	slog.Info("gRPC server ready", slog.String("listenAddr", *listenAddr))

	err = grpcServer.Serve(listener)
	if err != nil {
		slog.Error("gRPC server stopped with error", slog.Any("error", err))
	}
	return err
}
