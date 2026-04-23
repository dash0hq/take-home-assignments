import { Request, Response } from 'express';
import { IQueue } from '../queue/types';
import { logBatchSchema } from '../validation/log-ingestion-schema';
import { LogEntry } from '../queue/types';

export const createLogIngestionHandler = (logQueue: IQueue<LogEntry, any>) => {
  return async (req: Request, res: Response) => {
    const result = logBatchSchema.safeParse(req.body);

    if (!result.success) {
      return res.status(400).json({
        error: 'Invalid payload',
        details: result.error.issues,
      });
    }

    try {
      await logQueue.push(result.data as any);
      res.status(202).json({ message: 'Logs accepted and queued' });
    } catch (error) {
      console.error('Failed to push logs to queue', error);
      res.status(500).json({ error: 'Internal server error' });
    }
  };
};
