import {beforeEach, describe, expect, it, vi} from "vitest";
import { SlidingWindowRateLimiter } from "./sliding-window-rate-limiter";

describe('InMemoryRateLimiter', () => {
    let rateLimiter: SlidingWindowRateLimiter;
    const limit = 5;
    const windowMs = 1000;

    beforeEach(() => {
        rateLimiter = new SlidingWindowRateLimiter(limit, windowMs);
        vi.useFakeTimers();
    });

    it('should allow requests within the limit', async () => {
        for (let i = 0; i < limit; i++) {
            const isLimited = await rateLimiter.isRateLimited('test-key');
            expect(isLimited).toBe(false);
        }
    });

    it('should block requests exceeding the limit', async () => {
        for (let i = 0; i < limit; i++) {
            await rateLimiter.isRateLimited('test-key');
        }
        const isLimited = await rateLimiter.isRateLimited('test-key');
        expect(isLimited).toBe(true);
    });

    it('should reset after the window expires', async () => {
        for (let i = 0; i < limit; i++) {
            await rateLimiter.isRateLimited('test-key');
        }

        expect(await rateLimiter.isRateLimited('test-key')).toBe(true);

        // Advance time past the window
        vi.advanceTimersByTime(windowMs + 1);

        expect(await rateLimiter.isRateLimited('test-key')).toBe(false);
    });

    it('should track different keys separately', async () => {
        for (let i = 0; i < limit; i++) {
            await rateLimiter.isRateLimited('key-1');
        }

        expect(await rateLimiter.isRateLimited('key-1')).toBe(true);
        expect(await rateLimiter.isRateLimited('key-2')).toBe(false);
    });
});