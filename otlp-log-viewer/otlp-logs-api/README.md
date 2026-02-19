# Mock OTLP Logs API Server

> **Note:** This is the API server that provides mock OTLP logs data.
> Your task is to build a **separate frontend application** that consumes this API.
> Do not extend this codebase.

## Deployed Endpoint

The API is already deployed and available at:

```
https://take-home-assignment-otlp-logs-api.vercel.app/api/logs
```

You can use this endpoint directly in your application without running the server locally.

## Local Development (Optional)

If you want to run the API server locally:

```sh
npm install
npm run dev
```

The server will be available at `http://localhost:3000/api/logs`.

## API Details

- **Method:** GET
- **Response:** `IExportLogsServiceRequest` from `@opentelemetry/otlp-transformer`
- **Data:** Randomly generated mock logs (data varies each request)
- **CORS:** Enabled for all origins
