import { NodeSDK } from '@opentelemetry/sdk-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { ExpressInstrumentation } from '@opentelemetry/instrumentation-express';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';
import { trace, Tracer } from '@opentelemetry/api';
import { config } from '../config';

const exporter = new OTLPTraceExporter({
  url: config.otlpEndpoint,
  headers: config.dash0ApiKey ? {
    Authorization: `Bearer ${config.dash0ApiKey}`,
  } : {},
});

const sdk = new NodeSDK({
  traceExporter: exporter,
  instrumentations: [
    new HttpInstrumentation() as any,
    new ExpressInstrumentation() as any,
  ],
});

export const startTelemetry = () => {
  sdk.start();
  console.log('OpenTelemetry SDK started');
};

export const getTracer = (name: string): Tracer => {
  return trace.getTracer(name);
};

export const stopTelemetry = async () => {
  await sdk.shutdown();
  console.log('OpenTelemetry SDK stopped');
};
