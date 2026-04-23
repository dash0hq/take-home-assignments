import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { LogWorker } from './log-worker';
import { LogEntry, QueuedLogEntry } from '../services/queue/types';
import { IQueue } from '../services/queue/types';

// Mock config
vi.mock('../config', () => ({
  config: {
    concurrencyLimit: 2,
    processingDelayMs: 10,
    maxRetries: 3
  }
}));

// Mock telemetry to avoid errors and overhead
vi.mock('../telemetry/otel', () => ({
  getTracer: () => ({
    startActiveSpan: vi.fn((name, cb) => cb({
      setAttributes: vi.fn(),
      setStatus: vi.fn(),
      recordException: vi.fn(),
      end: vi.fn(),
    })),
  }),
}));

describe('LogWorker', () => {
  let logQueue: IQueue<LogEntry, QueuedLogEntry>;
  let dlq: IQueue<LogEntry, QueuedLogEntry>;
  let logWorker: LogWorker;

  beforeEach(() => {
    logQueue = {
      push: vi.fn(),
      pop: vi.fn(),
      pushBack: vi.fn(),
      length: vi.fn().mockReturnValue(0),
    };
    dlq = {
      push: vi.fn(),
      pop: vi.fn(),
      pushBack: vi.fn(),
      length: vi.fn().mockReturnValue(0),
    };
    logWorker = new LogWorker(logQueue, dlq);
    vi.useFakeTimers();
  });

  afterEach(() => {
    logWorker.stop();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  describe('processLogEntry', () => {
    it('should process a log entry successfully', async () => {
      const logEntry: QueuedLogEntry = {
        id: '1',
        timestamp: new Date().toISOString(),
        level: 'info',
        message: 'test message',
        retryCount: 0
      };

      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
      
      const processPromise = (logWorker as any).processLogEntry(logEntry);
      await vi.advanceTimersByTimeAsync(10); // processingDelayMs is 10
      await processPromise;

      expect(consoleSpy).toHaveBeenCalledWith(JSON.stringify(logEntry));
    });

    it('should retry a log entry if it contains FAIL', async () => {
      const logEntry: QueuedLogEntry = {
        id: '2',
        timestamp: new Date().toISOString(),
        level: 'error',
        message: 'THIS WILL FAIL',
        retryCount: 0
      };

      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      // Mock Math.random to have a predictable jitter (0.5 results in jitter = 0)
      const randomSpy = vi.spyOn(Math, 'random').mockReturnValue(0.5);
      
      const processPromise = (logWorker as any).processLogEntry(logEntry);
      
      // Advance for processing delay
      await vi.advanceTimersByTimeAsync(10);
      
      // Advance for backoff delay (2^1 * 1000 = 2000, jitter = 0 with random=0.5)
      await vi.advanceTimersByTimeAsync(2000);
      
      await processPromise;

      expect(logEntry.retryCount).toBe(1);
      expect(logQueue.pushBack).toHaveBeenCalledWith(logEntry);
      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Retrying log entry 2'));
      
      randomSpy.mockRestore();
    });

    it('should apply jitter to backoff delay', () => {
        // We can test the private method directly for logic verification
        const retryCount = 1;
        const baseDelay = 2000; // 1000 * 2^1
        
        // Mock random to -20% jitter (Math.random() * 2 - 1 = -1 when Math.random() = 0)
        vi.spyOn(Math, 'random').mockReturnValue(0);
        let delay = (logWorker as any).backoffDelay(retryCount);
        expect(delay).toBe(baseDelay * 0.8);
        
        // Mock random to +20% jitter (Math.random() * 2 - 1 = 1 when Math.random() = 1)
        vi.spyOn(Math, 'random').mockReturnValue(1);
        delay = (logWorker as any).backoffDelay(retryCount);
        expect(delay).toBe(baseDelay * 1.2);
        
        // Mock random to no jitter (Math.random() * 2 - 1 = 0 when Math.random() = 0.5)
        vi.spyOn(Math, 'random').mockReturnValue(0.5);
        delay = (logWorker as any).backoffDelay(retryCount);
        expect(delay).toBe(baseDelay);
        
        vi.restoreAllMocks();
    });

    it('should stop retrying after maxRetries and push to DLQ', async () => {
      const logEntry: QueuedLogEntry = {
        id: '3',
        timestamp: new Date().toISOString(),
        level: 'error',
        message: 'FAIL message',
        retryCount: 3 // already at maxRetries
      };

      const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      
      const processPromise = (logWorker as any).processLogEntry(logEntry);
      await vi.advanceTimersByTimeAsync(10);
      await processPromise;

      expect(logEntry.retryCount).toBe(3); 
      expect(logQueue.pushBack).not.toHaveBeenCalled();
      expect(dlq.pushBack).toHaveBeenCalledWith(logEntry);
      expect(errorSpy).toHaveBeenCalledWith(expect.stringContaining('failed after 3 retries. Moving to DLQ.'));
    });
  });

  describe('processLoop', () => {
    it('should process logs from queue when started', async () => {
      const logEntry: QueuedLogEntry = {
        id: '4',
        timestamp: new Date().toISOString(),
        level: 'info',
        message: 'queued log',
        retryCount: 0
      };

      let popped = false;
      (logQueue.pop as any).mockImplementation(() => {
        if (!popped) {
          popped = true;
          return Promise.resolve(logEntry);
        }
        return Promise.resolve(undefined);
      });

      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      logWorker.start();
      
      // Advance timers to trigger the loop and processing delay
      await vi.advanceTimersByTimeAsync(100); 

      expect(logQueue.pop).toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(JSON.stringify(logEntry));
    });

    it('should respect concurrency limit', async () => {
        // config.concurrencyLimit is 2
        const logEntry: QueuedLogEntry = {
          id: '5',
          timestamp: new Date().toISOString(),
          level: 'info',
          message: 'concurrent log',
          retryCount: 0
        };
  
        // Mock pop to return a log twice, then block
        let callCount = 0;
        (logQueue.pop as any).mockImplementation(async () => {
          if (callCount < 2) {
            callCount++;
            return logEntry;
          }
          // block further pops
          return new Promise(() => {});
        });
  
        vi.spyOn(console, 'log').mockImplementation(() => {});
        
        logWorker.start();
        
        // Wait for loop to try popping
        await vi.advanceTimersByTimeAsync(0);
        
        // Initially it should pop 2 logs and be at limit
        expect(callCount).toBe(2);

        // Advance a bit more, it shouldn't pop more because all workers are busy waiting for pop
        await vi.advanceTimersByTimeAsync(100);
        expect(callCount).toBe(2);
      });

      it('should prevent multiple pops while waiting for one to resolve', async () => {
        // This test simulates the concurrency bug by having a slow pop()
        let popInProgress = 0;
        let resolvePop: (value: QueuedLogEntry | undefined) => void;
        
        const popPromise = new Promise<QueuedLogEntry | undefined>((resolve) => {
          resolvePop = resolve;
        });

        (logQueue.pop as any).mockImplementation(() => {
          popInProgress++;
          return popPromise;
        });

        logWorker.start();

        // Let the loop run. It should fill all available slots up to concurrency limit (2 in this test)
        await vi.advanceTimersByTimeAsync(0);

        expect(popInProgress).toBe(2);

        // Even if more time passes, it shouldn't trigger another pop() because all loops are blocked at limit
        await vi.advanceTimersByTimeAsync(100);
        expect(popInProgress).toBe(2);

        // Resolve one
        resolvePop!({ id: '6', message: 'delayed', retryCount: 0, level: 'info', timestamp: '' });
        await vi.advanceTimersByTimeAsync(0);

        // One worker finished its pop, and should be processing now.
        // It might even start another pop if it finished processing immediately, 
        // but in this test we just want to verify we don't exceed the limit.
      });

      it('should stop processing when stop() is called', async () => {
        (logQueue.pop as any).mockReturnValue(new Promise(() => {})); // Never resolves
        
        logWorker.start();
        expect((logWorker as any).running).toBe(true);
        
        logWorker.stop();
        expect((logWorker as any).running).toBe(false);
      });
  });
});
