import { IRateLimiter } from './types';

export class SlidingWindowRateLimiter implements IRateLimiter {
  private requests: Map<string, number[]>;
  private readonly limit: number;
  private readonly windowMs: number;

  constructor(limit = 10, windowMs = 1000) {
    this.requests = new Map();
    this.limit = limit;
    this.windowMs = windowMs;
  }

  async isRateLimited(apiKey: string): Promise<boolean> {
    const now = Date.now();
    const timestamps = this.requests.get(apiKey) || [];

    // Remove expired timestamps from the front
    while (timestamps.length && timestamps[0] <= now - this.windowMs) {
      timestamps.shift();
    }

    if (timestamps.length >= this.limit) {
      return true;
    }

    timestamps.push(now);
    this.requests.set(apiKey, timestamps);
    return false;
  }
}
