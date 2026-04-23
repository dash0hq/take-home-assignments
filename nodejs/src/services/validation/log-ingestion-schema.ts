import { z } from 'zod';

export const logEntrySchema = z.object({
  timestamp: z.string().datetime(),
  level: z.enum(['info', 'warn', 'error', 'debug']),
  message: z.string().min(1),
  meta: z.record(z.string(), z.unknown()).optional(),
});

export const logBatchSchema = z.array(logEntrySchema);
