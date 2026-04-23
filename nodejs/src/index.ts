import { startTelemetry, stopTelemetry } from './telemetry/otel';
startTelemetry();

import { createApp } from './app';
import { config } from './config';
import { MemoryApiKeyProvider } from './services/auth/memory-api-key-provider';
import { MemoryQueue } from './services/queue/memory-queue';
import { SlidingWindowRateLimiter } from './services/rate-limit/sliding-window-rate-limiter';
import { LogWorker } from './worker/log-worker';

import { User } from './services/auth/types';

const initialUsers: User[] = [
    { id: 'user-1', name: 'Test User 1', tier: 'free' },
    { id: 'user-2', name: 'Test User 2', tier: 'pro' },
];

const initialKeys = config.apiKeys.map((key, index) => ({
    key,
    userId: index % 2 === 0 ? 'user-1' : 'user-2'
}));

const apiKeyProvider = new MemoryApiKeyProvider(initialUsers, initialKeys);
const rateLimiter = new SlidingWindowRateLimiter(config.rateLimit);
const logQueue = new MemoryQueue();
const dlq = new MemoryQueue();

const app = createApp(apiKeyProvider, rateLimiter, logQueue);
const worker = new LogWorker(logQueue, dlq);
worker.start();

const server = app.listen(config.port, () => {
    console.log(JSON.stringify({ level: 'info', message: `Server started`, port: config.port }));
});

process.on('SIGTERM', async () => {
    console.log('SIGTERM received, shutting down gracefully');
    
    // Set a timeout to force exit if graceful shutdown takes too long
    const shutdownTimeout = setTimeout(() => {
        console.error('Shutdown timed out, forcing exit');
        process.exit(1);
    }, 30000); // 30 seconds

    try {
        // 1. Stop accepting new requests
        server.close((err) => {
            if (err) {
                console.error('Error closing server:', err);
            } else {
                console.log('HTTP server closed');
            }
        });

        // 2. Stop worker and wait for active tasks to finish
        await worker.stop();
        console.log('Worker stopped and active tasks completed');

        // 3. Stop telemetry
        await stopTelemetry();
        
        clearTimeout(shutdownTimeout);
        console.log('Graceful shutdown completed');
        process.exit(0);
    } catch (error) {
        console.error('Error during graceful shutdown:', error);
        process.exit(1);
    }
});
