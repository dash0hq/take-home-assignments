# Event Tracking

## Introduction
This take-home assignment is designed to give you an opportunity to demonstrate your skills and experience in
building a web application that interacts with a database. We expect you to spend 3-5 hours on this assignment. If you find yourself spending more time
than that, please stop and submit what you have. We are not looking for a complete solution, but rather a demonstration
of your skills and experience.

To submit your solution, please create a public GitHub repository and send us the link. Please include a `README.md` file
with instructions on how to run your application.

## Overview
The goal of this assignment is to build a simple web application that renders a list of [log records](https://opentelemetry.io/docs/concepts/signals/logs/) from an OTLP logs
endpoint and to report usage statistics to a small backend. For an overview of the expected capabilities, see the [Expected Capabilities](#expected-capabilities) section
below. 

This assignment is a variation of our [OTLP Log Viewer](../otlp-log-viewer/README.md) assignment. Consequently, some
sections will refer to sections of the OTLP Log Viewer's assignment to avoid duplication.

## OTLP Logs HTTP Endpoint
See the [OTLP Logs HTTP Endpoint](../otlp-log-viewer/README.md#otlp-logs-http-endpoint) section of the OTLP Log Viewer's assignment.

## Expected Capabilities
- Retrieve the list of log records from the OTLP logs HTTP endpoint mentioned above (at runtime).
- Render the list of log records in a table with the following columns:
    - Severity
    - Time
    - Body
- When clicking a log record, expand the row and show all attributes associated with the log record.
- Record statistics about the user's interactions with the application and send them to a backend.
    - When a user opens the app.
    - When a user clicks on a log record.

## Technology Constraints
- Use React, TypeScript and Next.js with the app router for the UI.
- Use a separate backend, e.g., using [Express.js](https://expressjs.com/), to accept and persist the usage statistics.
- Use any additional libraries you want and need.

## Supporting Material
Within the [example-nodejs-postgres](./example-nodejs-postgres/README.md) directory, you will find an example that shows how to…

1. Start a PostgreSQL database using Docker.
2. Create a table using initialization scripts.
3. Query data from the table using Node.js.

## Notes
See the [Notes](../otlp-log-viewer/README.md#notes) section of the OTLP Log Viewer's assignment. Also…

## References
See the [References](../otlp-log-viewer/README.md#references) section of the OTLP Log Viewer's assignment.
