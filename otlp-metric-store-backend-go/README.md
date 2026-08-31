# OTLP Metric Storage (Go)

## Introduction
This take-home assignment is designed to give you an opportunity to demonstrate your skills and experience in
building a small backend application. We expect you to spend 3-4 hours on this assignment (using AI coding agents).
If you find yourself spending more time than that, please stop and submit what you have. We are not looking for a
complete solution, but rather a demonstration of your skills and experience.

To submit your solution, please create a public GitHub repository and send us the link. Please include a `README.md` file
with instructions on how to run your application.

## Overview
The goal of this assignment is to build a simple backend application that receives [metric datapoints](https://opentelemetry.io/docs/concepts/signals/metrics/)
on a gRPC endpoint and processes them, before storing in ClickHouse.
Current state is that we have a gRPC endpoint for receiving metrics, and Gauge and Sum type get correctly converted to
records and inserted into ClickHouse. This is tested with both unit- and integration-tests.

What we're looking for is to extract meta-data about the metrics into a separate table, which will then act as a 'lookup'
table, and that actual data-points just get stored as value + timestamp and with a reference to the lookup table.

Think about and keep in mind the following things:
- How to do the reference between tables?
- How to efficiently store the meta-data in ClickHouse?
- All data should be stored in such a way that full table scans are never needed, under the assumption data always gets queried for a specific time-frame
- Other than time-frame, there are no other mandatory filters for querying
- While you can assume cardinality of the metrics is 'low', e.g. Resources (Attributes) are likely to change over time 

Your solution should take into account high throughput, both in number of messages and the number of metrics / data-points per message.

Feel free to use the existing scaffoling in this folder. Of course, you can also change anything else as you see fit.

## Technology Constraints
- Your Go program should compile using standard Go SDK, and be compatible with Go 1.26.
- Use any additional libraries you want and need.

## Notes
- As this assignment is for the role of a Staff / Senior Product Engineer, we expect you to pay some attention to maintainability and operability of the solution. For example:
  - Consistent terminology usage
  - Validation of the behaviour
  - Include signals / events to help in debugging
- Assume that this application will be deployed to production. Build it accordingly.

## Usage

Build the application:
```shell
go build ./...
```

Run the application:
```shell
go run ./...
```

Run tests
```shell
go test ./...
```

## Assumptions

- **Metric metadata cardinality is low relative to data-point volume.**
  A relatively small number of metric series produces a much larger number of
  data points. Storing shared metadata separately therefore reduces significant
  duplication.

- **Changes to identifying metadata create a new metric series.**
  If the resource, scope, metric type, or attributes that identify a series change,
  the new combination is treated as a separate series.

- **Existing metric tables may have readers outside this service.**
  Dashboards, alerting systems, ad-hoc queries, or other services may depend on
  the current ClickHouse tables. To avoid disrupting unknown consumers, the
  existing schemas are left unchanged.

- **The normalized implementation currently covers Gauge and Sum.**
  These are the metric types supported by the existing ingestion path. The same
  approach can be extended to the remaining OTLP metric types later.


## Design Decisions

### Backward-compatible schema evolution

The existing metric tables remain unchanged.

The normalized model is introduced through new tables:

- `otel_metrics_metadata`
- `otel_metrics_gauge_v2`
- `otel_metrics_sum_v2`

During migration, the service writes to both the existing and normalized schemas.

This allows the new design to be introduced and validated without forcing
existing consumers to migrate immediately.

Once downstream consumers have moved to the normalized schema, the legacy write
path can be retired.


### Metadata normalization

The existing schema repeats resource, scope, metric, and attribute metadata on
every data point.

The new model separates:

- **series-level metadata**, stored once and referenced through `MetadataID`
- **data-point values**, stored separately

This reduces repeated storage while preserving the relationship between a metric
series and its data points.


### Metric identity

`MetadataID` represents a distinct metric series.

The identity is based on the fields that materially distinguish one series from
another, including:

- `MetricType`
- `MetricName`
- `MetricUnit`
- `ResourceAttributes`
- `ScopeName`
- `ScopeVersion`
- `ScopeAttributes`
- `Attributes`
- `AggregationTemporality` for Sum metrics
- `IsMonotonic` for Sum metrics

The following fields do not create a new series identity:

- `ResourceSchemaUrl`
- `ScopeSchemaUrl`
- `ScopeDroppedAttrCount`
- `MetricDescription`
- `StartTimeUnix`
- `TimeUnix`
- `Value`
- `Flags`

For example, a Gauge and a Sum with the same metric name are still different
metric series and therefore require separate identities.

Fields that describe or observe a metric, such as metric description, timestamps,
values, and flags, do not create a new series identity.

`MetadataID` is generated deterministically from the metric identity.

The implementation uses `city.Hash64`.


### Query model

The normalized model is intended to support retrieving metric series over a time
range.

> **Product / query-semantics note**
>
> Before finalizing the production query model, I would validate the expected
> workflow with Product and downstream consumers.
>
> A time range alone is unlikely to be sufficient for most meaningful metric
> queries. Users will normally also need to identify what they want to observe,
> for example through service name, metric name, metric type, attributes, or a
> previously selected metric series.
>
> These answers would help determine the most appropriate production query and
> storage model.


### Series cache

An in-memory series cache reduces unnecessary repeated metadata writes for
frequently observed metric series.

The cache is an optimization only and is not required for correctness.

A restart or multiple service replicas may result in some additional metadata
writes, but data-point ingestion continues normally.


## Migration Strategy

The migration follows an expand-and-migrate approach:

    Existing schema
          |
          v
    Add normalized schema
          |
          v
    Dual-write old + new
          |
          v
    Validate normalized data
          |
          v
    Migrate consumers
          |
          v
    Stop legacy writes
          |
          v
    Retire legacy schema

This assignment implements the normalized schema and the migration write path.

The final consumer migration, rollback period, and retirement of the legacy
schema would depend on the production rollout strategy.


## Known Limitations / Open Items

### MetadataID collision handling

The current 64-bit hash-based identifier does not provide a mathematical
uniqueness guarantee.

A production system requiring strict collision-safe identity should use a wider
identifier or introduce explicit collision handling.


### Production deployment configuration

The standalone server still requires production ClickHouse configuration before
it can operate as a complete deployable service.


### Legacy-table retirement

Dual-writing is intended as a temporary migration mechanism.

The exact cutover point and retention period for the legacy tables depend on
downstream consumer migration and rollout strategy.


### Remaining metric types

Histogram, ExponentialHistogram, and Summary continue to use the existing schema.

The normalization approach can be extended to them in a future iteration.


## Testing

The implementation includes unit and integration tests covering:

- metadata identity generation
- Gauge and Sum identity separation
- metadata and data-point mapping
- cache behavior
- backward compatibility with existing writes
- normalized schema creation and persistence
- end-to-end gRPC ingestion through ClickHouse
- metadata update and deduplication behavior

## References

- [OpenTelemetry Metrics](https://opentelemetry.io/docs/concepts/signals/metrics/)
- [OpenTelemetry Protocol (OTLP)](https://github.com/open-telemetry/opentelemetry-proto)
