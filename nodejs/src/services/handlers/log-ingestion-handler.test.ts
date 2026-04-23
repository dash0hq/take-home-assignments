import { describe, it, expect, beforeEach, vi } from 'vitest';
import express from 'express';
import request from 'supertest';
import { createLogIngestionHandler } from './log-ingestion-handler';
import { IQueue } from '../queue/types';
import { LogEntry } from '../queue/types';

describe('LogIngestionHandler', () => {
  let app: express.Express;
  let mockLogQueue: IQueue<LogEntry, any>;

  beforeEach(() => {
    app = express();
    app.use(express.json());

    mockLogQueue = {
      push: vi.fn(),
      pop: vi.fn(),
      pushBack: vi.fn(),
      length: vi.fn().mockReturnValue(0),
    };

    const handler = createLogIngestionHandler(mockLogQueue);
    app.post('/logs/json', handler);
  });

  it('should return 400 when payload is invalid', async () => {
    const invalidPayload = [{ timestamp: 'invalid-date', level: 'info', message: '' }];
    
    const response = await request(app)
      .post('/logs/json')
      .send(invalidPayload);

    expect(response.status).toBe(400);
    expect(response.body.error).toBe('Invalid payload');
    expect(response.body.details).toBeDefined();
    expect(Array.isArray(response.body.details)).toBe(true);
  });

  it('should return 202 and queue logs when payload is valid', async () => {
    const validPayload = [{ 
      timestamp: new Date().toISOString(), 
      level: 'info', 
      message: 'Test message' 
    }];
    
    const response = await request(app)
      .post('/logs/json')
      .send(validPayload);

    expect(response.status).toBe(202);
    expect(response.body.message).toBe('Logs accepted and queued');
    expect(mockLogQueue.push).toHaveBeenCalled();
  });
});
