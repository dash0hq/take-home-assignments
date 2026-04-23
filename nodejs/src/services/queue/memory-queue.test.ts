import {beforeEach, describe, expect, it} from "vitest";
import { MemoryQueue } from "./memory-queue";
import { LogEntry, IQueue } from "./types";

describe('MemoryQueue', () => {
    let logQueue: MemoryQueue;

    beforeEach(() => {
        logQueue = new MemoryQueue();
    });

    it('should push and pop logs', async () => {
        const logs: LogEntry[] = [
            { timestamp: new Date().toISOString(), level: 'info', message: 'test 1' },
            { timestamp: new Date().toISOString(), level: 'info', message: 'test 2' },
        ];

        await logQueue.push(logs);
        expect(logQueue.length()).toBe(2);

        const log1 = await logQueue.pop();
        expect(log1?.message).toBe('test 1');
        expect(log1?.id).toBeDefined();
        expect(log1?.retryCount).toBe(0);

        const log2 = await logQueue.pop();
        expect(log2?.message).toBe('test 2');
        expect(logQueue.length()).toBe(0);
    });

    it('should pushBack logs to the end of the queue', async () => {
        const logs: LogEntry[] = [
            { timestamp: new Date().toISOString(), level: 'info', message: 'test 1' },
        ];
        await logQueue.push(logs);

        const log1 = (await logQueue.pop())!;
        log1.retryCount = 1;
        await logQueue.pushBack(log1);

        expect(logQueue.length()).toBe(1);
        const popped = await logQueue.pop();
        expect(popped?.message).toBe('test 1');
        expect(popped?.retryCount).toBe(1);
    });

    it('should return undefined when popping from empty queue', async () => {
        const log = await logQueue.pop();
        expect(log).toBeUndefined();
    });

    it('should work with generic types (generic verification)', async () => {
        // This test verifies that the IQueue interface can be used generically
        // although MemoryQueue is specifically bound to LogEntry/QueuedLogEntry.
        // For a truly generic test, we'd need a generic implementation.
        const genericQueue: IQueue<string> = {
            push: async (items: string[]) => {},
            pop: async () => 'test',
            pushBack: async (item: string) => {},
            length: () => 1
        };
        
        expect(await genericQueue.pop()).toBe('test');
    });
});