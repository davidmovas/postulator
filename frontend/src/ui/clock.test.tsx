import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useElapsed, useNow } from "./clock.js";

const start = new Date("2026-10-03T12:00:00Z").getTime();

beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
});

afterEach(() => {
    vi.useRealTimers();
});

describe("useNow", () => {
    it("ticks on its interval while it is active", () => {
        const { result } = renderHook(() => useNow(1000, true));
        expect(result.current).toBe(start);
        act(() => {
            vi.advanceTimersByTime(999);
        });
        expect(result.current).toBe(start);
        act(() => {
            vi.advanceTimersByTime(1);
        });
        expect(result.current).toBe(start + 1000);
        act(() => {
            vi.advanceTimersByTime(2000);
        });
        expect(result.current).toBe(start + 3000);
    });

    it("holds still while it is not active and resumes when it is", () => {
        const { result, rerender } = renderHook(({ active }) => useNow(1000, active), {
            initialProps: { active: false },
        });
        act(() => {
            vi.advanceTimersByTime(5000);
        });
        expect(result.current).toBe(start);
        rerender({ active: true });
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(result.current).toBe(start + 6000);
    });

    it("stops its timer when it unmounts", () => {
        const { unmount } = renderHook(() => useNow(1000, true));
        expect(vi.getTimerCount()).toBe(1);
        unmount();
        expect(vi.getTimerCount()).toBe(0);
    });
});

describe("useElapsed", () => {
    it("counts whole seconds while running and rests at zero otherwise", () => {
        const { result, rerender } = renderHook(({ running }) => useElapsed(running), {
            initialProps: { running: false },
        });
        expect(result.current).toBe(0);
        act(() => {
            vi.advanceTimersByTime(3000);
        });
        expect(result.current).toBe(0);
        rerender({ running: true });
        act(() => {
            vi.advanceTimersByTime(2000);
        });
        expect(result.current).toBe(2);
        rerender({ running: false });
        expect(result.current).toBe(0);
        expect(vi.getTimerCount()).toBe(0);
    });

    it("starts again from zero on the next run", () => {
        const { result, rerender } = renderHook(({ running }) => useElapsed(running), {
            initialProps: { running: true },
        });
        act(() => {
            vi.advanceTimersByTime(4000);
        });
        expect(result.current).toBe(4);
        rerender({ running: false });
        rerender({ running: true });
        expect(result.current).toBe(0);
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(result.current).toBe(1);
    });
});
