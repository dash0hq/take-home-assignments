export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warn' | 'error' | 'debug';
  message: string;
  meta?: Record<string, unknown>;
}

export interface QueuedLogEntry extends LogEntry {
  id: string;
  retryCount: number;
}

export interface IQueue<TIn, TOut = TIn> {
  push(items: TIn[]): Promise<void>;
  pop(): Promise<TOut | undefined>;
  pushBack(item: TOut): Promise<void>;
  length(): number;
}
