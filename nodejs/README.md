# Node.js - take home assignment

## Background
You're building a core piece of an observability platform: a **log ingestion service**. 
Clients (agents running on customer infrastructure) ship batches of structured log entries to your service over HTTP. 
Your service must authenticate those clients, buffer incoming logs asynchronously, and process them reliably — 
all while being designed with scale in mind.

## Requirements

### Auth & Security

* Clients authenticate using an API key passed as a Bearer token.
  * API keys can be stored in memory for this exercise, but design your code to allow swapping in a persistent store later.
* Implement a simple rate limiter per API key (e.g. max 10 requests/second). 
  * An in-memory implementation is fine

### Log Ingestion Endpoint

* `POST /logs/json` - Accepts a batch of log entries as a JSON array of objects and each object is a log record.
* Validate the payload
* Push them onto an internal queue

#### Example payload
```json
[
    {
      "timestamp": "2024-11-01T12:00:00Z",
      "level": "error",
      "message": "Disk usage above 90%",
      "meta": { "host": "prod-server-1", "service": "disk-monitor" }
    }
]
```

### Log Processing
* A background worker drains the queue and "processes" each log entry
  * They should be logged to STDOUT
  * Simulate a processing delay per log entry to mimic real-world conditions
* The worker should process entries concurrently, but with a configurable concurrency limit (e.g. max 5 at a time).
* If a log entry fails processing, it should be retried up to 3 times

### Observability
* Instrument the service using the OpenTelemetry Node.js SDK (@opentelemetry/sdk-node)
* Create a child span for each log entry when it is picked up and processed by the worker.
* Add relevant span attributes: log.level, log.service (from meta.service), queue.depth, worker.retry_count.
* Mark spans as error when processing fails, and record the exception.
* Use Dash0 as Backend for OpenTelemetry traces. You can sign up for a free account at https://dash0.com and get your API key to configure the OpenTelemetry exporter.
