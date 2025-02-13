package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	collectorlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

type LogProcessor struct {
	attributeKey   string
	windowDuration time.Duration
	mu             sync.RWMutex
	counts         map[string]int
	windowStart    time.Time
}

// {Post Assignment Thoughts}
// I went with a simple map for aggregating entries.
// It is worth pointing out the possibility of a map exceeding the allocated memory
// of a given server (OOM), given the current windowDuration setup, resulting in lost data.
// We would need to know more about the server specs and throughput for optimal bounderies,
// but preventing 10k-logs/s with a windowDuration of 10 hours is probably a good start.
// In addition to this, it would make sense to have a maxMapSize const that takes server specs
// or optimal data inserts into account, processing and restarting the window early if needed.
func NewLogProcessor(attributeKey string, windowDuration time.Duration) *LogProcessor {
	p := &LogProcessor{
		attributeKey:   attributeKey,
		windowDuration: windowDuration,
		counts:         make(map[string]int),
		windowStart:    time.Now(),
	}
	go p.processWindows()

	return p
}

func (p *LogProcessor) processWindows() {
	ticker := time.NewTicker(p.windowDuration)

	// defer ticker.stop() is not needed as of GO 1.23.
	// We keep it for good measure, as this code
	// could be running on ealier versions.
	defer ticker.Stop()

	for range ticker.C {
		p.printAndResetCounts()
	}
}

func (p *LogProcessor) printAndResetCounts() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Given the terminal output is currently being populated
	// by a lot of information every N seconds, I decided
	// to write to a file, making the viewing of the output easier.
	//
	// Opening a file every N seconds has a lot of overhead costs compared to wrtiting to stdout,
	// this is purely for making a more pleasant experience for the user of this code.
	// Ideally this data is sent and stored somewhere properly.
	//
	// If a file was to be used, opening the file when starting the service
	// would decrease costs significantly depending on the windowDuration.
	// This assumes that we have a single service, as we would not be able to properly
	// sync the content of the files across services.
	if len(p.counts) > 0 {
		f, err := os.OpenFile("log_summary.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			os.Interrupt.Signal()
			return
		}
		defer f.Close()

		fmt.Fprintf(f, "\nLog Summary\n")
		fmt.Fprintf(f, "attributeKey = \"%s\"\n", *attributeKey)
		for k, v := range p.counts {
			fmt.Fprintf(f, "Value: \"%s\"\tCount: %d\n", k, v)
		}
		fmt.Fprintf(f, "\n")
	}

	// Reset values
	p.counts = make(map[string]int)
	p.windowStart = time.Now()
}

func (p *LogProcessor) ProcessLogs(ctx context.Context, request *collectorlogspb.ExportLogsServiceRequest) error {
	_, span := tracer.Start(ctx, "ProccessLogs")
	defer span.End()

	span.AddEvent("Aquirring lock")
	p.mu.Lock()
	span.AddEvent("Got lock")

	defer func() {
		span.AddEvent("Unlocking")
		defer p.mu.Unlock()
	}()

	// {Post Assignment Thoughts}:
	// I decided to go with the simplest approach to start with, 3 loops, and see if time allowed
	// me to iterate (it did not).
	//
	// This is O(n^3) and will proably make most developers second-guess their existence.
	// Should a ResourceLog not contain a magnitute of elements at each level,
	// then this simple solution might suffice. I doubt that is the case given the amount
	// of data/logs such a service will receive, so spending more time on this
	// and collaborating with a colleague would be ideal as this is the heart of the service.
	//
	// When writing this logic, certain assumptions were made:
	// - Attribute keys are NOT invalid:
	//		- "" (empty string)
	//		- Special characters (spaces, none-utf8, ect..)
	// - Attribute values will NOT be nil
	// - Attribute keys and values will NOT exceed a maximum lenght limit.
	//
	// This was naive thinking and should be checked and tested for, ideally before wrtiting
	// the solution, to ensure such edge-cases are properly dealt with.
	for _, resourceLogs := range request.GetResourceLogs() {
		resourceValue := findAttribute(ctx, resourceLogs.Resource.GetAttributes(), p.attributeKey)
		if resourceValue != "" {
			p.counts[resourceValue]++
		}

		for _, scopeLogs := range resourceLogs.GetScopeLogs() {
			scopeValue := findAttribute(ctx, scopeLogs.Scope.GetAttributes(), p.attributeKey)
			if scopeValue != "" {
				p.counts[scopeValue]++
			}

			for _, log := range scopeLogs.GetLogRecords() {
				logValue := findAttribute(ctx, log.GetAttributes(), p.attributeKey)
				if logValue != "" {
					p.counts[logValue]++
					continue
				}

				p.counts["unknown"]++
			}
		}
	}

	return nil
}

func findAttribute(ctx context.Context, attrs []*commonpb.KeyValue, key string) string {
	_, span := tracer.Start(ctx, "findAttribute")
	defer span.End()

	for _, attr := range attrs {
		if attr.GetKey() == key {
			attrAnyVal := attr.GetValue()
			switch attrAnyVal.GetValue().(type) {
			case *commonpb.AnyValue_StringValue:
				return attrAnyVal.GetStringValue()
			case *commonpb.AnyValue_IntValue:
				return fmt.Sprintf("%d", attrAnyVal.GetIntValue())
			case *commonpb.AnyValue_DoubleValue:
				return fmt.Sprintf("%g", attrAnyVal.GetDoubleValue())
			case *commonpb.AnyValue_BoolValue:
				return fmt.Sprintf("%t", attrAnyVal.GetBoolValue())
			}
		}
	}
	return ""
}

// /\___(o)> (Meow, I'm a duck)
// \_____/
