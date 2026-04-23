import { describe, it, expect } from 'vitest';
import { logEntrySchema } from './log-ingestion-schema';

describe('logEntrySchema', () => {
  it('should validate successfully with various meta types', () => {
    const validLog = {
      timestamp: new Date().toISOString(),
      level: 'info',
      message: 'Test message',
      meta: { 
        host: 'localhost', 
        service: 'test-service',
        count: 123,
        active: true,
        extra: { foo: 'bar' }
      }
    };
    const result = logEntrySchema.safeParse(validLog);
    expect(result.success).toBe(true);
  });

  it('should validate successfully when meta is missing', () => {
    const logWithoutMeta = {
      timestamp: new Date().toISOString(),
      level: 'info',
      message: 'Test message'
    };
    const result = logEntrySchema.safeParse(logWithoutMeta);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.meta).toBeUndefined();
    }
  });

  it('should fail if timestamp is missing', () => {
    const invalidLog = {
      level: 'info',
      message: 'Test message'
    };
    const result = logEntrySchema.safeParse(invalidLog);
    expect(result.success).toBe(false);
  });
});
