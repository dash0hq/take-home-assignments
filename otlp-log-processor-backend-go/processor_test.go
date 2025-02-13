package main

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	collectorlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

// {Post Assignment Thoughts}
// I call this "happy path" testing, it's a great confidence booster to ensure you are on the right
// path, but I would rather have invested the time in edge-case tests. If the solution would not be
// working on a base level as expected, odds are I would have seen that being the case while
// coding/testing my soltion via the terminal as I go, given the scale of this task.
// I can not say the same for the edge-cases.
func TestLogProcessor(t *testing.T) {
	t.Run("proccess attributes at different levels", func(t *testing.T) {
		processor := NewLogProcessor("foo", time.Minute)
		req := &collectorlogspb.ExportLogsServiceRequest{
			ResourceLogs: []*logspb.ResourceLogs{
				{
					Resource: &resourcepb.Resource{
						Attributes: []*commonpb.KeyValue{
							{
								Key: "foo",
								Value: &commonpb.AnyValue{
									Value: &commonpb.AnyValue_StringValue{
										StringValue: "resource_value",
									},
								},
							},
						},
					},

					ScopeLogs: []*logspb.ScopeLogs{
						{
							Scope: &commonpb.InstrumentationScope{
								Attributes: []*commonpb.KeyValue{
									{
										Key: "foo",
										Value: &commonpb.AnyValue{
											Value: &commonpb.AnyValue_StringValue{
												StringValue: "scope_value",
											},
										},
									},
								},
							},
							LogRecords: []*logspb.LogRecord{
								{
									Attributes: []*commonpb.KeyValue{
										{
											Key: "foo",
											Value: &commonpb.AnyValue{
												Value: &commonpb.AnyValue_StringValue{
													StringValue: "log_value",
												},
											},
										},
									},
								},
								{
									Attributes: []*commonpb.KeyValue{},
								},
							},
						},
					},
				},
			},
		}

		err := processor.ProcessLogs(context.Background(), req)
		assert.NoError(t, err)

		processor.mu.RLock()
		defer processor.mu.RUnlock()

		assert.Equal(t, 1, processor.counts["resource_value"], "resouce_value")
		assert.Equal(t, 1, processor.counts["scope_value"], "scope_value")
		assert.Equal(t, 1, processor.counts["log_value"], "log_value")
		assert.Equal(t, 1, processor.counts["unknown"], "unknown")
	})

	t.Run("handles different attribute value types", func(t *testing.T) {
		processor := NewLogProcessor("test_key", time.Minute)
		req := &collectorlogspb.ExportLogsServiceRequest{
			ResourceLogs: []*logspb.ResourceLogs{
				{
					Resource: &resourcepb.Resource{},
					ScopeLogs: []*logspb.ScopeLogs{
						{
							LogRecords: []*logspb.LogRecord{
								{
									Attributes: []*commonpb.KeyValue{
										{
											Key: "test_key",
											Value: &commonpb.AnyValue{
												Value: &commonpb.AnyValue_IntValue{
													IntValue: 666,
												},
											},
										},
									},
								},
								{
									Attributes: []*commonpb.KeyValue{
										{
											Key: "test_key",
											Value: &commonpb.AnyValue{
												Value: &commonpb.AnyValue_BoolValue{
													BoolValue: true,
												},
											},
										},
									},
								},
								{
									Attributes: []*commonpb.KeyValue{
										{
											Key: "test_key",
											Value: &commonpb.AnyValue{
												Value: &commonpb.AnyValue_StringValue{
													StringValue: "never_gonna_give_you_up",
												},
											},
										},
									},
								},
								{
									Attributes: []*commonpb.KeyValue{
										{
											Key: "test_key",
											Value: &commonpb.AnyValue{
												Value: &commonpb.AnyValue_DoubleValue{
													DoubleValue: 66.66,
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
		}

		err := processor.ProcessLogs(context.Background(), req)
		assert.NoError(t, err)

		processor.mu.RLock()
		defer processor.mu.RUnlock()

		assert.Equal(t, 1, processor.counts["666"], "IntValue")
		assert.Equal(t, 1, processor.counts["never_gonna_give_you_up"], "StringValue")
		assert.Equal(t, 1, processor.counts["true"], "BoolValue")
		assert.Equal(t, 1, processor.counts["66.66"], "DoubleValue")
	})

	t.Run("window duration resets counts", func(t *testing.T) {
		windowDuration := 100 * time.Millisecond
		processor := NewLogProcessor("foo", windowDuration)
		req := &collectorlogspb.ExportLogsServiceRequest{
			ResourceLogs: []*logspb.ResourceLogs{
				{
					Resource: &resourcepb.Resource{
						Attributes: []*commonpb.KeyValue{
							{
								Key: "foo",
								Value: &commonpb.AnyValue{
									Value: &commonpb.AnyValue_StringValue{
										StringValue: "test_value",
									},
								},
							},
						},
					},
				},
			},
		}

		err := processor.ProcessLogs(context.Background(), req)
		assert.NoError(t, err)

		processor.mu.RLock()
		assert.Equal(t, 1, processor.counts["test_value"], "test_value")
		processor.mu.RUnlock()

		time.Sleep(windowDuration + 50*time.Millisecond)

		processor.mu.RLock()
		assert.Equal(t, 0, len(processor.counts), "processor counts should have been reset")
		processor.mu.RUnlock()
	})
}

// I did not get to this part during the allocated time.
// Some obvious test cases would include:
// - Check for an empty ExportLogsServiceRequst
// - Check for nil values in any layers of the ExportLogsServiceRequest.
// - Missing attributes at various levels
func TestLogProcessorEdgeCases(t *testing.T) {
	// testCases := []struct {
	// 	name        string
	// 	setupReq    func() *collectorlogspb.ExportLogsServiceRequest
	// 	expectCount map[string]int
	// 	expectErr   bool
	// }{}
}

// {Post Assignment Thoughts}:
// Looking at this again, I think the test missed its mark.
// The ResourceLog is too shallow and each slice does not contain enough elements
// to give proper insights into the potential downside of the current {ProcessLogs} implementation.
// I do not know how many logs each scope Dash0 expect to get in general, I suspect that it
// varies a lot from client to client, but ideally the benchmark would at least reflect the
// client causing the most stress on our system, both in throughput and object size, to ensure that
// we do not run into issues with the current load.
func BenchmarkLogProcessor(b *testing.B) {
	createLogRecord := func(value string) *logspb.LogRecord {
		if value == "" {
			return &logspb.LogRecord{
				Attributes: []*commonpb.KeyValue{},
			}
		}

		return &logspb.LogRecord{
			Attributes: []*commonpb.KeyValue{
				{
					Key: "foo",
					Value: &commonpb.AnyValue{
						Value: &commonpb.AnyValue_StringValue{
							StringValue: value,
						},
					},
				},
			},
		}
	}

	createBatchRequest := func(n int) *collectorlogspb.ExportLogsServiceRequest {
		req := &collectorlogspb.ExportLogsServiceRequest{
			ResourceLogs: []*logspb.ResourceLogs{
				{
					Resource: &resourcepb.Resource{
						Attributes: []*commonpb.KeyValue{
							{
								Key: "foo",
								Value: &commonpb.AnyValue{
									Value: &commonpb.AnyValue_StringValue{
										StringValue: "resource_value",
									},
								},
							},
						},
					},
					ScopeLogs: []*logspb.ScopeLogs{
						{
							Scope:      &commonpb.InstrumentationScope{},
							LogRecords: make([]*logspb.LogRecord, n),
						},
					},
				},
			},
		}

		for i := 0; i < n; i++ {
			switch {
			case i%4 == 0:
				req.ResourceLogs[0].ScopeLogs[0].LogRecords[i] = createLogRecord("value_" + strconv.Itoa(i))
			case i%4 == 1:
				req.ResourceLogs[0].ScopeLogs[0].LogRecords[i] = createLogRecord("")
			default:
				req.ResourceLogs[0].ScopeLogs[0].LogRecords[i] = createLogRecord("common_value")
			}
		}

		return req
	}

	benchCases := []struct {
		name     string
		logCount int
	}{
		{"Batch_100", 100},
		{"Batch_1.000", 1000},
		{"Batch_10.000", 10000},
		{"Batch_100.000", 100000},
		{"Batch_1.000.000", 1000000},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			processor := NewLogProcessor("foo", time.Hour)
			req := createBatchRequest(bc.logCount)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := processor.ProcessLogs(ctx, req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
