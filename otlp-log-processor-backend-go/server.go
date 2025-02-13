package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var (
	listenAddr            = flag.String("listenAddr", "localhost:4317", "The listen address")
	maxReceiveMessageSize = flag.Int("maxReceiveMessageSize", 16777216, "The max message size in bytes the server can receive")
	attributeKey          = flag.String("attributeKey", "foo", "The attribute key to count unique values for")
	windowDuration        = flag.Duration("windowDuration", 5*time.Second, "The duration of the counting window")
)

const name = "dash0.com/otlp-log-processor-backend"

var (
	tracer              = otel.Tracer(name)
	meter               = otel.Meter(name)
	logger              = otelslog.NewLogger(name)
	logsReceivedCounter metric.Int64Counter
)

func init() {
	var err error
	logsReceivedCounter, err = meter.Int64Counter(
		"com.dash0.homeexercise.logs.received",
		metric.WithDescription("The number of logs received by otlp-log-processor-backend"),
		metric.WithUnit("{log}"),
	)
	if err != nil {
		panic(err)
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

// {Post Assignment Thoughts - Overview}:
// This was a very fun take-home assignment, so thank you to Jochen Schalanda and whomever helped
// create it.
//
// Throughtout the the code you will see {Post Assignment Thoughts} comments. These are self-analysing
// comments on a second readthrough after the allocated time. Hopefully you will find these
// PAT comments as usefull as I found them fun to write.
//
// The solution consists of 3 major parts:
// 1. Gracefull shutdown of the service via signals
// 2. Capturing and aggregating logs based on given attribute key and timespan
// 3. Test and benchamarking
//
// Following the 80/20 rule I would say this solution is 80% done, which still leaves us with
// all the timeconsuming and important parts yet to be tackled, one of which is debugability of the
// system. While I am happy with the work done I am regretting not diving deeper into telemetry, given
// the nature of Dash0's domain.
func run() (err error) {
	slog.SetDefault(logger)
	logger.Info("Starting application")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	flag.Parse()

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.MaxRecvMsgSize(*maxReceiveMessageSize),
		grpc.Creds(insecure.NewCredentials()),
	)

	processor := NewLogProcessor(*attributeKey, *windowDuration)
	collogspb.RegisterLogsServiceServer(grpcServer, newServer(*listenAddr, processor))

	// Enable reflection for easy grpcurl testing via terminal.
	// Use/edit the "log_request.json" file for easy testing.
	// The output will be available in "log_summary.txt".
	reflection.Register(grpcServer)

	logger.Debug("Starting gRPC server")
	// Ensure we always have memeroy available for a signal.
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- grpcServer.Serve(listener)
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting server.
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	grpcServer.GracefulStop()
	return
}
