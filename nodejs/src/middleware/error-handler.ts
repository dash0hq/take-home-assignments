import { Request, Response, NextFunction } from 'express';

export const errorHandlerMiddleware = (
    error: Error,
    _req: Request,
    res: Response,
    _next: NextFunction
): void => {
    console.error('Unhandled error:', error);
    res.status(500).json({ error: 'Internal server error' });
};