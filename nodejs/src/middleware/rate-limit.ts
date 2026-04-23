import { Request, Response, NextFunction } from 'express';
import { IRateLimiter } from '../services/rate-limit/types';

export const createRateLimitMiddleware = (rateLimiter: IRateLimiter) => {
  return async (req: Request, res: Response, next: NextFunction) => {
    const user = req.user;
    if (!user) {
      return res.status(500).json({ error: 'User not found in request context' });
    }

    const isLimited = await rateLimiter.isRateLimited(user.id);
    if (isLimited) {
      return res.status(429).json({ error: 'Rate limit exceeded' });
    }

    next();
  };
};
