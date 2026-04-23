import dotenv from 'dotenv';
dotenv.config();

export const config = {
    port: parseInt(process.env.PORT ?? '3003', 10),
    apiKeys: (process.env.API_KEYS ?? 'test-key-1,test-key-2').split(','),
    rateLimit: parseInt(process.env.RATE_LIMIT ?? '10', 10),
    concurrencyLimit: parseInt(process.env.CONCURRENCY_LIMIT ?? '5', 10),
    processingDelayMs: parseInt(process.env.PROCESSING_DELAY_MS ?? '100', 10),
    maxRetries: parseInt(process.env.MAX_RETRIES ?? '3', 10),
    otlpEndpoint: process.env.OTLP_ENDPOINT ?? 'https://ingest.dash0.com/otlp',
    dash0ApiKey: process.env.DASH0_API_KEY,
};