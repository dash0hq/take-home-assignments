import {describe, it, expect, beforeEach, vi} from 'vitest';
import express from 'express';
import request from 'supertest';
import {createAuthMiddleware} from './auth';
import {createRateLimitMiddleware} from './rate-limit';
import {IApiKeyProvider,} from '../services/auth/types';
import {IRateLimiter} from '../services/rate-limit/types';


describe('RateLimitMiddleware', () => {
    let app: express.Express;
    let mockRateLimiter: IRateLimiter;
    let mockApiKeyProvider: IApiKeyProvider;

    beforeEach(() => {
        app = express();
        app.use(express.json());

        mockApiKeyProvider = {
            getUserByApiKey: vi.fn().mockResolvedValue({id: 'user-1', name: 'User 1', tier: 'free'}),
            addKey: vi.fn(),
            removeKey: vi.fn(),
            listKeys: vi.fn(),
        };

        mockRateLimiter = {
            isRateLimited: vi.fn(),
        };

        const authMiddleware = createAuthMiddleware(mockApiKeyProvider);
        const rateLimitMiddleware = createRateLimitMiddleware(mockRateLimiter);

        app.post('/test', authMiddleware, rateLimitMiddleware, (_req, res) => {
            res.status(200).json({success: true});
        });
    });

    it('should allow request when not rate limited', async () => {
        vi.mocked(mockRateLimiter.isRateLimited).mockResolvedValue(false);

        const response = await request(app)
            .post('/test')
            .set('Authorization', 'Bearer valid-key');

        expect(response.status).toBe(200);
        expect(response.body).toEqual({success: true});
    });

    it('should block request when rate limited', async () => {
        vi.mocked(mockRateLimiter.isRateLimited).mockResolvedValue(true);

        const response = await request(app)
            .post('/test')
            .set('Authorization', 'Bearer valid-key');

        expect(response.status).toBe(429);
        expect(response.body).toEqual({error: 'Rate limit exceeded'});
    });
});