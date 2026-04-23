import { LogEntry, QueuedLogEntry, IQueue } from './types';
import { v7 as uuidv7 } from 'uuid';

export class MemoryQueue implements IQueue<LogEntry, QueuedLogEntry> {
  private queue: QueuedLogEntry[] = [];

  async push(logs: LogEntry[]): Promise<void> {
    const queuedLogs: QueuedLogEntry[] = logs.map(log => ({
      ...log,
      id: uuidv7(),
      retryCount: 0
    }));
    this.queue.push(...queuedLogs);
  }

  async pop(): Promise<QueuedLogEntry | undefined> {
    return this.queue.shift();
  }

  length(): number {
    return this.queue.length;
  }

  async pushBack(log: QueuedLogEntry): Promise<void> {
    this.queue.push(log);
  }
}
