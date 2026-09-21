import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { clearToasts, dismissToast, holdToastMs, pushToast, readToasts } from "./toasts.js";

beforeEach(() => {
    vi.useFakeTimers();
    clearToasts();
});

afterEach(() => {
    vi.useRealTimers();
});

describe("toasts", () => {
    it("takes a plain confirmation away by itself", () => {
        pushToast("info", "Saved");
        expect(readToasts()).toHaveLength(1);
        vi.advanceTimersByTime(holdToastMs);
        expect(readToasts()).toHaveLength(0);
    });

    it("keeps one that carries an action until it is answered", () => {
        const id = pushToast("warning", "Tor Browser is not installed.", null, { label: "Set up", to: "/settings/browser" });
        vi.advanceTimersByTime(holdToastMs * 4);
        expect(readToasts()).toHaveLength(1);
        dismissToast(id);
        expect(readToasts()).toHaveLength(0);
    });

    it("does not take away a toast that was dismissed and replaced", () => {
        const first = pushToast("info", "Saved");
        dismissToast(first);
        pushToast("info", "Deleted");
        vi.advanceTimersByTime(holdToastMs - 1);
        expect(readToasts().map((held) => held.message)).toEqual(["Deleted"]);
    });
});
