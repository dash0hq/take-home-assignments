# OTLP Log Parser (Go)

## Introduction
This take-home assignment is designed to give you an opportunity to demonstrate your skills and experience in
building a small backend application. We expect you to spend 3-4 hours on this assignment. If you find yourself spending more time
than that, please stop and submit what you have. We are not looking for a complete solution, but rather a demonstration
of your skills and experience.

To submit your solution, please create a public GitHub repository and send us the link. Please include a `README.md` file
with instructions on how to run your application.

## Overview
The goal of this assignment is to build a simple backend application that receives [log records](https://opentelemetry.io/docs/concepts/signals/logs/)
on a gRPC endpoint and processes them. Based on a **configurable attribute key and duration**, the application has to keep
counts of the number of unique log records per distinct attribute value. And within each window (configurable duration) print /
log these counts to stdout.
Note that the configurable attribute may appear either on Resource, Scope or Log level.

Pseudo example:
- "my log body 1" - {"foo":"bar", "baz":"qux"}
- "my log body 2" - {"foo":"qux", "baz":"qux"}
- "my log body 3" - {"baz":"qux"}
- "my log body 4" - {"foo":"baz"}
- "my log body 5" - {"foo":"baz", "baz":"qux"}

For example for configured attribute key "foo" it should report:
- "bar" - 1
- "qux" - 1
- "baz" - 2
- unknown - 1

Your solution should take into account high throughput, both in number of messages and the number of records per message.

Feel free to use the existing scaffoling in this folder, for example by fleshing out the implementation of the `Export`
method in `logs_service.go`. Of course, you can also change anything else as you see fit.

## Technology Constraints
- Your Go program should compile using standard Go SDK, and be compatible with Go 1.23.
- Use any additional libraries you want and need.

## Notes
- As this assignment is for the role of a Senior Product Engineer, we expect you to pay some attention to maintainability and operability of the solution. For example:
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

- [OpenTelemetry Logs](https://opentelemetry.io/docs/concepts/signals/logs/)
- [OpenTelemetry Protocol (OTLP)](https://github.com/open-telemetry/opentelemetry-proto)
- [OTLP Logs Examples](https://github.com/open-telemetry/opentelemetry-proto/blob/main/examples/logs.json)
