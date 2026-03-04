package main

import (
	"context"
	"log"
	"net"
	"testing"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otelmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestMetricsServiceServer_Export(t *testing.T) {
	ctx := context.Background()

	client, closer := server()
	defer closer()

	type expectation struct {
		out *colmetricspb.ExportMetricsServiceResponse
		err error
	}

	tests := map[string]struct {
		in       *colmetricspb.ExportMetricsServiceRequest
		expected expectation
	}{
		"Must_Success": {
			in: &colmetricspb.ExportMetricsServiceRequest{
				ResourceMetrics: []*otelmetrics.ResourceMetrics{
					{
						ScopeMetrics: []*otelmetrics.ScopeMetrics{},
						SchemaUrl:    "dash0.com/otlp-metrics-processor-backend",
					},
				},
			},
			expected: expectation{
				out: &colmetricspb.ExportMetricsServiceResponse{},
				err: nil,
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			out, err := client.Export(ctx, tt.in)
			if err != nil {
				if tt.expected.err.Error() != err.Error() {
					t.Errorf("Err -> \nWant: %q\nGot: %q\n", tt.expected.err, err)
				}
			} else {
				expectedPartialSuccess := tt.expected.out.GetPartialSuccess()
				if expectedPartialSuccess.GetRejectedDataPoints() != out.GetPartialSuccess().GetRejectedDataPoints() ||
					expectedPartialSuccess.GetErrorMessage() != out.GetPartialSuccess().GetErrorMessage() {
					t.Errorf("Out -> \nWant: %q\nGot : %q", tt.expected.out, out)
				}
			}

		})
	}
}

func server() (colmetricspb.MetricsServiceClient, func()) {
	addr := "localhost:4317"
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	baseServer := grpc.NewServer()
	colmetricspb.RegisterMetricsServiceServer(baseServer, newServer(addr, nil))
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	closer := func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}

	client := colmetricspb.NewMetricsServiceClient(conn)

	return client, closer
}
