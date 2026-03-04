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

## References

- [OpenTelemetry Metrics](https://opentelemetry.io/docs/concepts/signals/metrics/)
- [OpenTelemetry Protocol (OTLP)](https://github.com/open-telemetry/opentelemetry-proto)
