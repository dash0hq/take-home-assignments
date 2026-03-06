import express, { Request, Response } from 'express';

export function createApp() {
    const app = express();

    app.use(express.json());

    app.get('/', (_req: Request, res: Response) => {
        res.json({ message: 'Hello World' });
    });

    return app;
}