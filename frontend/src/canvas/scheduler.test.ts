import { describe, expect, it } from "vitest";

import { frameScheduler } from "./scheduler.js";

function fakeFrames() {
    const pending = new Map<number, () => void>();
    let next = 1;
    return {
        request: (callback: () => void): number => {
            const handle = next;
            next += 1;
            pending.set(handle, callback);
            return handle;
        },
        cancel: (handle: number): void => {
            pending.delete(handle);
        },
        run(): void {
            const callbacks = [...pending.values()];
            pending.clear();
            for (const callback of callbacks) {
                callback();
            }
        },
        get queued(): number {
            return pending.size;
        },
    };
}

describe("frameScheduler", () => {
    it("coalesces several requests into one frame", () => {
        const frames = fakeFrames();
        let drawn = 0;
        const scheduler = frameScheduler(
            () => {
                drawn += 1;
            },
            frames.request,
            frames.cancel,
        );
        scheduler.request();
        scheduler.request();
        scheduler.request();
        expect(frames.queued).toBe(1);
        frames.run();
        expect(drawn).toBe(1);
        scheduler.request();
        expect(frames.queued).toBe(1);
        frames.run();
        expect(drawn).toBe(2);
    });

    it("cancels a pending frame", () => {
        const frames = fakeFrames();
        let drawn = 0;
        const scheduler = frameScheduler(
            () => {
                drawn += 1;
            },
            frames.request,
            frames.cancel,
        );
        scheduler.request();
        scheduler.cancel();
        frames.run();
        expect(drawn).toBe(0);
        expect(frames.queued).toBe(0);
    });
});
