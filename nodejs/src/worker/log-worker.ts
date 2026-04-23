import { LogEntry, QueuedLogEntry } from '../services/queue/types';
import { IQueue } from '../services/queue/types';
import { config } from '../config';
import { getTracer } from '../telemetry/otel';
import { SpanStatusCode } from '@opentelemetry/api';

const tracerName = 'log-worker';
const activeSpanName = 'process-log-entry';

export class LogWorker {
  private logQueue: IQueue<LogEntry, QueuedLogEntry>;
  private dlq: IQueue<LogEntry, QueuedLogEntry>;
  private running = false;
  private workers: Promise<void>[] = [];
  private tracer = getTracer(tracerName);

  constructor(
    logQueue: IQueue<LogEntry, QueuedLogEntry>,
    dlq: IQueue<LogEntry, QueuedLogEntry>
  ) {
    this.logQueue = logQueue;
    this.dlq = dlq;
  }

  start() {
    this.running = true;
    for (let i = 0; i < config.concurrencyLimit; i++) {
      this.workers.push(this.processLoop());
    }
  }

  async stop() {
    this.running = false;
    await Promise.all(this.workers);
    this.workers = [];
  }

  private async processLoop() {
    while (this.running) {
      try {
        const logEntry = await this.logQueue.pop();
        if (!logEntry) {
          // If stopped while waiting for pop, don't sleep
          if (!this.running) break;
          await sleep(100);
          continue;
        }
        
        await this.processLogEntry(logEntry);

      } catch (error) {
        console.error('Worker error:', error);
      }
    }
  }

  private async processLogEntry(logEntry: QueuedLogEntry) {
    await this.tracer.startActiveSpan(activeSpanName, async (span) => {
      try {
        span.setAttributes({
          'log.level': logEntry.level,
          'log.service': (logEntry.meta?.service as string) ?? 'unknown',
          'queue.depth': this.logQueue.length(),
          'worker.retry_count': logEntry.retryCount,
        });

        // Simulate processing delay
        await sleep(config.processingDelayMs);

        // Randomly fail to test retries (e.g., 10% failure rate)
        // For this exercise, let's keep it simple and just log to stdout
        // But the requirement says "If a log entry fails processing, it should be retried"
        // So I'll simulate a failure here
        
        // Let's say it fails if the message contains "FAIL"
        if (logEntry.message.includes('FAIL')) {
            throw new Error('Simulated processing failure');
        }

        console.log(JSON.stringify(logEntry));
        span.setStatus({ code: SpanStatusCode.OK });
      } catch (error: any) {
        span.recordException(error);
        span.setStatus({
          code: SpanStatusCode.ERROR,
          message: error.message,
        });

        await this.handleRetry(logEntry);
      } finally {
        span.end();
      }
    });
  }

  private async handleRetry(logEntry: QueuedLogEntry) {
    if (logEntry.retryCount >= config.maxRetries) {
      console.error(
          `Log entry ${logEntry.id} failed after ${logEntry.retryCount} retries. Moving to DLQ.`
      );
      await this.dlq.pushBack(logEntry);
      return;
    }

    logEntry.retryCount++;

    console.warn(`Retrying log entry ${logEntry.id} (attempt ${logEntry.retryCount})`);
    
    await sleep(this.backoffDelay(logEntry.retryCount));

    await this.logQueue.pushBack(logEntry);
  }

  private backoffDelay(retryCount: number) {
    const baseDelay = Math.min(1000 * 2 ** retryCount, 10_000);
    // Add jitter: ±20% of the base delay
    const jitter = baseDelay * 0.2 * (Math.random() * 2 - 1);
    return Math.max(0, baseDelay + jitter);
  }
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}