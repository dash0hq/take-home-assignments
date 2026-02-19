# OTLP Log Viewer

## Quick Start

| | |
|---|---|
| **Time** | 3-4 hours (submit what you have) |
| **Stack** | React, TypeScript, Next.js (App Router) |
| **Submit** | Public GitHub repo + README |
| **Bonus** | Vercel deployment |

> **Create a NEW repository.** Do not extend this codebase.

---

## Part 1: Coding Assignment

### Scenario

Your team needs a web application to visualize [OTLP log records](https://opentelemetry.io/docs/concepts/signals/logs/) from a backend service. Engineers should be able to quickly scan logs, drill into details, and understand log distribution patterns across services.

### Your Task

Build a log viewer that fetches data from the provided API endpoint and addresses these requirements:

1. **Log List View** — Display logs in a table (Severity, Time, Body) with expandable rows showing all attributes
2. **Histogram** — Visualize log distribution over time (X: Time, Y: Count)
3. **Group by Service** — Add a toggle that switches between flat list view and grouped view (organized by parent resource with collapsible groups)

### What We're Looking For

- Data fetching and state management patterns
- Data transformation for nested OTLP types
- Component architecture and TypeScript usage
- UI/UX consistent with observability domain conventions
- Production-ready code organization

---

## Part 2: Interview Discussion (Do Not Code)

### Scenario

Users of the log viewer have started asking for filtering capabilities. The product brief is intentionally vague:

> "We need a way for users to filter logs and share interesting findings with teammates."

### Your Task

As the Senior Product Engineer (Frontend), you're asked to help shape the approach before implementation begins.

During the interview, walk us through:

1. **Clarifying the problem** — What questions would you ask product, backend, and users?
2. **Structuring the solution** — How would you design the UI and frontend architecture?
3. **Identifying trade-offs** — What key decisions would you make, and what are the pros and cons?

### What We're Looking For

- Ability to clarify ambiguity in product requirements
- Structural thinking about UI architecture, components, and data flow
- Articulating trade-offs (complexity vs. maintainability, performance vs. UX)
- Bonus: Observability-aware thinking

---

## API Reference

**Endpoint:**
```
GET https://take-home-assignment-otlp-logs-api.vercel.app/api/logs
```

**TypeScript Types:**
```typescript
import {
  IExportLogsServiceRequest,
  IResourceLogs,
  ILogRecord,
} from "@opentelemetry/otlp-transformer";
```

**Data Structure:**
```
IExportLogsServiceRequest
└── resourceLogs[]
    ├── resource.attributes[]
    │   ├── service.name
    │   ├── service.namespace
    │   └── service.version
    └── scopeLogs[]
        └── logRecords[]
            ├── timeUnixNano
            ├── severityText / severityNumber
            ├── body
            └── attributes[]
```

> The API generates random mock data on each request.

---

## References

- [OpenTelemetry Logs Concepts](https://opentelemetry.io/docs/concepts/signals/logs/)
- [OTLP Protocol](https://github.com/open-telemetry/opentelemetry-proto)
- [OTLP Logs Example JSON](https://github.com/open-telemetry/opentelemetry-proto/blob/main/examples/logs.json)
