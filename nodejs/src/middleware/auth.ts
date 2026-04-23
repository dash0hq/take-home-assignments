import { Request, Response, NextFunction } from 'express';
import { IApiKeyProvider } from '../services/auth/types';
import { User } from '../services/auth/types';

declare global {
  namespace Express {
    interface Request {
      user?: User;
    }
  }
}

export const createAuthMiddleware = (apiKeyProvider: IApiKeyProvider) => {
  return async (req: Request, res: Response, next: NextFunction) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'Missing or invalid Authorization header' });
    }

    const apiKey = authHeader.split(' ')[1];
    const user = await apiKeyProvider.getUserByApiKey(apiKey);

    if (!user) {
      return res.status(401).json({ error: 'Invalid API key' });
    }

    req.user = user;
    next();
  };
};
