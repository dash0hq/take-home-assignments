import { IApiKeyProvider, User } from './types';

export class MemoryApiKeyProvider implements IApiKeyProvider {
  private keysToUserIds: Map<string, string> = new Map();
  private users: Map<string, User> = new Map();

  constructor(initialUsers: User[], initialKeys: { key: string; userId: string }[]) {
    for (const user of initialUsers) {
      this.users.set(user.id, user);
    }
    for (const { key, userId } of initialKeys) {
      this.keysToUserIds.set(key, userId);
    }
  }

  async getUserByApiKey(apiKey: string): Promise<User | undefined> {
    const userId = this.keysToUserIds.get(apiKey);
    if (!userId) return undefined;
    return this.users.get(userId);
  }

  async addKey(apiKey: string, userId: string): Promise<void> {
    this.keysToUserIds.set(apiKey, userId);
  }

  async removeKey(apiKey: string): Promise<void> {
    this.keysToUserIds.delete(apiKey);
  }

  async listKeys(userId?: string): Promise<string[]> {
    if (userId) {
      return Array.from(this.keysToUserIds.entries())
        .filter(([_, uid]) => uid === userId)
        .map(([key]) => key);
    }
    return Array.from(this.keysToUserIds.keys());
  }

  // Helper for tests/initialization to add users
  addUser(user: User): void {
    this.users.set(user.id, user);
  }
}
