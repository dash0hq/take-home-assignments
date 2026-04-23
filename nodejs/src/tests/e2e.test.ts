import request from 'supertest';
import { createApp } from '../app';
import { MemoryApiKeyProvider } from '../services/auth/memory-api-key-provider';
import { MemoryQueue } from '../services/queue/memory-queue';
import { SlidingWindowRateLimiter } from '../services/rate-limit/sliding-window-rate-limiter';
import { LogWorker } from '../worker/log-worker';
import { describe, it, expect, beforeEach, vi, afterAll, afterEach } from 'vitest';
import { User } from "../services/auth/types";

describe('System Requirements Verification (Integration)', () => {
    const apiKey = 'test-key-1';
    const initialUsers = [{ id: 'user-1', name: 'Test User 1', tier: 'free' }];
    const initialKeys = [{ key: apiKey, userId: 'user-1' }];
    
    let app: any;
    let logQueue: any;
    let dlq: any;
    let worker: any;

    beforeEach(() => {
        const apiKeyProvider = new MemoryApiKeyProvider(initialUsers as User[], initialKeys);
        const rateLimiter = new SlidingWindowRateLimiter(10); // 10 req/s
        logQueue = new MemoryQueue();
        dlq = new MemoryQueue();
        app = createApp(apiKeyProvider, rateLimiter, logQueue);
        worker = new LogWorker(logQueue, dlq);
    });

    it('should complete the full lifecycle: Auth -> Ingestion -> Processing', async () => {
        // 1. Ingest log via API (End-to-end through app)
        const response = await request(app)
            .post('/logs/json')
            .set('Authorization', `Bearer ${apiKey}`)
            .send([{ 
                timestamp: new Date().toISOString(), 
                level: 'info', 
                message: 'Integration test log',
                meta: { service: 'verif-service' }
            }]);
        
        expect(response.status).toBe(202);
        expect(logQueue.length()).toBe(1);

        // 2. Process log via worker
        const logEntry = await logQueue.pop();
        
        // Mock console.log to verify processing
        const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
        
        await (worker as any).processLogEntry(logEntry);
        
        expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('Integration test log'));
        expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('verif-service'));
        consoleSpy.mockRestore();
    });

    it('should enforce the rate limit (Integration check)', async () => {
        // This confirms the middleware and rate limiter work together in the app context
        for (let i = 0; i < 10; i++) {
            await request(app)
                .post('/logs/json')
                .set('Authorization', `Bearer ${apiKey}`)
                .send([{ timestamp: new Date().toISOString(), level: 'info', message: 'test' }]);
        }
        
        const response = await request(app)
            .post('/logs/json')
            .set('Authorization', `Bearer ${apiKey}`)
            .send([{ timestamp: new Date().toISOString(), level: 'info', message: 'test' }]);
        
        expect(response.status).toBe(429);
    });
});
