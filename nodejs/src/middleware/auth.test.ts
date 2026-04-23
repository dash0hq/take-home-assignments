import { describe, it, expect, beforeEach, vi } from 'vitest';
import express from 'express';
import request from 'supertest';
import { createAuthMiddleware } from './auth';
import { IApiKeyProvider } from '../services/auth/types';

describe('AuthMiddleware', () => {
  let app: express.Express;
  let mockApiKeyProvider: IApiKeyProvider;

  beforeEach(() => {
    app = express();
    app.use(express.json());

    mockApiKeyProvider = {
      getUserByApiKey: vi.fn(),
      addKey: vi.fn(),
      removeKey: vi.fn(),
      listKeys: vi.fn(),
    };

    const authMiddleware = createAuthMiddleware(mockApiKeyProvider);

    app.post('/test', authMiddleware, (_req, res) => {
      res.status(200).json({ success: true });
    });
  });

  it('should allow request with valid bearer token', async () => {
    vi.mocked(mockApiKeyProvider.getUserByApiKey).mockResolvedValue({
      id: 'user-123',
      name: 'Test User',
      tier: 'free'
    });

    const response = await request(app)
      .post('/test')
      .set('Authorization', 'Bearer valid-key');

    expect(response.status).toBe(200);
    expect(response.body).toEqual({ success: true });
    expect(mockApiKeyProvider.getUserByApiKey).toHaveBeenCalledWith('valid-key');
  });

  it('should block request when authorization header is missing', async () => {
    const response = await request(app).post('/test');

    expect(response.status).toBe(401);
    expect(response.body).toEqual({ error: 'Missing or invalid Authorization header' });
  });

  it('should block request when API key is invalid', async () => {
    vi.mocked(mockApiKeyProvider.getUserByApiKey).mockResolvedValue(undefined);

    const response = await request(app)
      .post('/test')
      .set('Authorization', 'Bearer invalid-key');

    expect(response.status).toBe(401);
    expect(response.body).toEqual({ error: 'Invalid API key' });
  });
});

