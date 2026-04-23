export interface IRateLimiter {
  isRateLimited(apiKey: string): Promise<boolean>;
}
