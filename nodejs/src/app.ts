import express, { Request, Response } from 'express';
import { LogEntry, QueuedLogEntry } from './services/queue/types';
import { IApiKeyProvider } from './services/auth/types';
import { IQueue } from './services/queue/types';
import { IRateLimiter } from './services/rate-limit/types';
import { createAuthMiddleware } from './middleware/auth';
import { createRateLimitMiddleware } from './middleware/rate-limit';
import { createLogIngestionHandler } from './services/handlers/log-ingestion-handler';
import { errorHandlerMiddleware } from './middleware/error-handler';

export function createApp(
  apiKeyProvider: IApiKeyProvider,
  rateLimiter: IRateLimiter,
  logQueue: IQueue<LogEntry, QueuedLogEntry>
) {
    const app = express();

    app.use(express.json());

    const authMiddleware = createAuthMiddleware(apiKeyProvider);
    const rateLimitMiddleware = createRateLimitMiddleware(rateLimiter);

    app.post(
      '/logs/json',
      authMiddleware,
      rateLimitMiddleware,
      createLogIngestionHandler(logQueue)
    );

    app.get('/', (_req: Request, res: Response) => {
        res.json({ message: 'Log Ingestion Service is running' });
    });

    app.use(errorHandlerMiddleware);

    return app;
}
