# OTLP Log Viewer

## Introduction
This take-home assignment is designed to give you an opportunity to demonstrate your skills and experience in
building a web application. We expect you to spend 3-4 hours on this assignment. If you find yourself spending more time
than that, please stop and submit what you have. We are not looking for a complete solution, but rather a demonstration
of your skills and experience.

To submit your solution, please create a public GitHub repository and send us the link. Please include a `README.md` file
with instructions on how to run your application. Bonus points for a deployment to Vercel.

## Overview
The goal of this assignment is to build a simple web application that renders a list of [log records](https://opentelemetry.io/docs/concepts/signals/logs/) from an OTLP logs
endpoint. For an overview of the expected capabilities, see the [Expected Capabilities](#expected-capabilities) section
below.

## OTLP Logs HTTP Endpoint
The OTLP logs HTTP endpoint is available at https://take-home-assignment-otlp-logs-api.vercel.app/api/logs. The endpoint
returns an OTLP logs data structure. You will use this endpoint as a data source within the assignment.

You will most likely need TypeScript types for the OTLP logs data structure:

```typescript
import {IExportLogsServiceRequest, IResourceLogs, ILogRecord} from "@opentelemetry/otlp-transformer";
```

## Expected Capabilities
 - Retrieve the list of log records from the OTLP logs HTTP endpoint mentioned above (at runtime).
 - Render the list of log records in a table with the following columns:
   - Severity
   - Time
   - Body
 - When clicking a log record, expand the row and show all attributes associated with the log record.
 - Render a histogram visualizing the distribution of log records.
   - X-Axis: Time
   - Y-Axis: Count

## Technology Constraints
 - Use React, TypeScript and Next.js with the app router.
 - Use any additional libraries you want and need.

## Notes
 - As this assignment is for the role of a Senior Product Engineer, we expect you to pay some attention to the experience and design of the solution. For example:
   - Structure consistent with established solutions in the observability domain
   - Consistent terminology usage
   - Some styling/visuals
 - You are not meant to extend the Next.js app residing within this repository. Please create a new Next.js app in a 
   separate repository.
 - Assume that this application will be deployed to production. Build it accordingly.

## References

- [OpenTelemetry Logs](https://opentelemetry.io/docs/concepts/signals/logs/)
- [OpenTelemetry Protocol (OTLP)](https://github.com/open-telemetry/opentelemetry-proto)
- [OTLP Logs Examples](https://github.com/open-telemetry/opentelemetry-proto/blob/main/examples/logs.json)