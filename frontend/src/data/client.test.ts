import { describe, expect, test } from "vitest";

import type { Code } from "../lib/errors.js";
import { announcementOf, quietMeta, quietOf } from "./client.js";
import { messages } from "./errors.js";

function rejection(code: Code, message = "", extra: Record<string, unknown> = {}): unknown {
    return new Error("rejected", { cause: { code, message, ...extra } });
}

function quiet(...codes: Code[]): ReadonlySet<Code> {
    return quietOf(quietMeta(codes));
}

const nothing = quietOf(undefined);

describe("the codes a caller declares it expects", () => {
    test("reads the set a query or a mutation declared", () => {
        expect(quiet("NOT_FOUND", "CONFLICT")).toEqual(new Set(["NOT_FOUND", "CONFLICT"]));
    });

    test("is empty when nothing was declared", () => {
        expect(nothing.size).toBe(0);
        expect(quietOf({}).size).toBe(0);
    });

    test("ignores a meta entry that is not a list of codes", () => {
        expect(quietOf({ quiet: "NOT_FOUND" }).size).toBe(0);
        expect(quietOf({ quiet: ["NOT_FOUND", 7, "NOPE"] })).toEqual(new Set(["NOT_FOUND"]));
    });
});

describe("what the caches announce", () => {
    test("raises the refetch toast for an undeclared NOT_FOUND", () => {
        expect(announcementOf(rejection("NOT_FOUND"), nothing)).toEqual({
            kind: "toast",
            tone: "info",
            message: messages.NOT_FOUND,
            afterMs: null,
        });
    });

    test("stays silent for a NOT_FOUND the caller expects", () => {
        expect(announcementOf(rejection("NOT_FOUND"), quiet("NOT_FOUND"))).toEqual({ kind: "none" });
    });

    test("stays silent for a CONFLICT the caller renders itself", () => {
        expect(announcementOf(rejection("CONFLICT"), quiet("CONFLICT"))).toEqual({ kind: "none" });
        expect(announcementOf(rejection("CONFLICT"), nothing)).toEqual({
            kind: "toast",
            tone: "info",
            message: messages.CONFLICT,
            afterMs: null,
        });
    });

    test("declaring one code leaves the others alone", () => {
        expect(announcementOf(rejection("NOT_FOUND"), quiet("CONFLICT"))).toEqual({
            kind: "toast",
            tone: "info",
            message: messages.NOT_FOUND,
            afterMs: null,
        });
    });

    test("never toasts a form error, which has an owner by definition", () => {
        expect(announcementOf(rejection("INVALID", "the sheet has no usable rows"), nothing)).toEqual({
            kind: "none",
        });
    });

    test("never toasts a field error", () => {
        const thrown = rejection("INVALID", "the path must start with a slash", {
            details: { field: "path" },
        });
        expect(announcementOf(thrown, nothing)).toEqual({ kind: "none" });
    });

    test("stays silent for a cancelled call", () => {
        const cancelled = Object.assign(new Error("cancelled"), { name: "CancelError" });
        expect(announcementOf(cancelled, nothing)).toEqual({ kind: "none" });
        expect(announcementOf(rejection("CANCELLED"), nothing)).toEqual({ kind: "none" });
    });

    test("swaps the tree for LOCKED even when the caller declared it", () => {
        expect(announcementOf(rejection("LOCKED"), quiet("LOCKED"))).toEqual({ kind: "locked" });
    });

    test("carries the retry delay on a throttle", () => {
        expect(announcementOf(rejection("RATE_LIMITED", "slow down", { retry: { afterMs: 4000 } }), nothing)).toEqual({
            kind: "toast",
            tone: "warning",
            message: messages.RATE_LIMITED,
            afterMs: 4000,
        });
    });

    test("warns for the credentials, the budget, the review and the external failure", () => {
        for (const code of ["UNAUTHORIZED", "BUDGET_EXCEEDED", "NEEDS_HUMAN", "EXTERNAL"] as const) {
            expect(announcementOf(rejection(code), nothing)).toEqual({
                kind: "toast",
                tone: "warning",
                message: messages[code],
                afterMs: null,
            });
        }
    });

    test("shouts for INTERNAL", () => {
        expect(announcementOf(rejection("INTERNAL"), nothing)).toEqual({
            kind: "toast",
            tone: "danger",
            message: messages.INTERNAL,
            afterMs: null,
        });
    });
});
