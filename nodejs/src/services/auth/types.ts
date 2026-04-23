// In real implementation the User model shall go into its own domain and not to sit in the auth service
export interface User {
  id: string;
  name: string;
  tier: 'free' | 'pro' | 'enterprise';
}

export interface IApiKeyProvider {
  getUserByApiKey(apiKey: string): Promise<User | undefined>;
  addKey(apiKey: string, userId: string): Promise<void>;
  removeKey(apiKey: string): Promise<void>;
  listKeys(userId?: string): Promise<string[]>;
}
