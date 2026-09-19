import { describe, expect, test } from "vitest";

import type { Code } from "../lib/errors.js";
import { failure, messages, needsPlugin, pluginCodeOf, react } from "./errors.js";

function rejection(code: Code, message = "", extra: Record<string, unknown> = {}): unknown {
    return new Error("rejected", { cause: { code, message, ...extra } });
}

describe("the error reaction map", () => {
    test("stays silent for a cancelled call", () => {
        const cancelled = Object.assign(new Error("cancelled"), { name: "CancelError" });
        expect(react(cancelled)).toEqual({ kind: "silent" });
        expect(react(rejection("CANCELLED"))).toEqual({ kind: "silent" });
    });

    test("swaps the tree for LOCKED", () => {
        expect(react(rejection("LOCKED"))).toEqual({ kind: "unlock" });
    });

    test("turns INVALID with a field into a field error", () => {
        expect(react(rejection("INVALID", "the path must start with a slash", { details: { field: "path" } }))).toEqual(
            { kind: "field", field: "path", message: "the path must start with a slash" },
        );
    });

    test("turns INVALID without a field into a form error", () => {
        expect(react(rejection("INVALID", "the sheet has no usable rows"))).toEqual({
            kind: "form",
            message: "the sheet has no usable rows",
        });
    });

    test("turns RATE_LIMITED into a countdown that honours retry.afterMs", () => {
        expect(react(rejection("RATE_LIMITED", "slow down", { retry: { afterMs: 4000 } }))).toEqual({
            kind: "throttle",
            afterMs: 4000,
            message: messages.RATE_LIMITED,
        });
    });

    test("floors a tiny retry delay", () => {
        const reaction = react(rejection("RATE_LIMITED", "slow down", { retry: { afterMs: 10 } }));
        expect(reaction).toEqual({ kind: "throttle", afterMs: 250, message: messages.RATE_LIMITED });
    });

    test("refetches and explains for NOT_FOUND and CONFLICT", () => {
        expect(react(rejection("NOT_FOUND"))).toEqual({ kind: "refetch", message: messages.NOT_FOUND });
        expect(react(rejection("CONFLICT"))).toEqual({ kind: "refetch", message: messages.CONFLICT });
    });

    test("raises the credentials banner for UNAUTHORIZED", () => {
        expect(react(rejection("UNAUTHORIZED"))).toEqual({
            kind: "credentials",
            message: messages.UNAUTHORIZED,
        });
    });

    test("names the budget, the review and the external failure", () => {
        expect(react(rejection("BUDGET_EXCEEDED"))).toEqual({ kind: "budget", message: messages.BUDGET_EXCEEDED });
        expect(react(rejection("NEEDS_HUMAN"))).toEqual({ kind: "review", message: messages.NEEDS_HUMAN });
        expect(react(rejection("EXTERNAL"))).toEqual({ kind: "external", message: messages.EXTERNAL });
    });

    test("uses the frozen message for INTERNAL because details are stripped", () => {
        expect(react(rejection("INTERNAL", "panic: nil map write", { details: { stack: "leaked" } }))).toEqual({
            kind: "fatal",
            message: messages.INTERNAL,
        });
    });

    test("treats an unrecognised rejection as INTERNAL", () => {
        expect(react(new Error("no cause at all"))).toEqual({ kind: "fatal", message: messages.INTERNAL });
    });
});

describe("a site that cannot issue a preview", () => {
    test("reads the plugin code off an INVALID refusal", () => {
        const missing = rejection("INVALID", "the plugin is not installed", { details: { code: "plugin_missing" } });
        const outdated = rejection("INVALID", "too old", { details: { code: "plugin_outdated", capability: "preview" } });
        expect(pluginCodeOf(failure(missing))).toBe("plugin_missing");
        expect(pluginCodeOf(failure(outdated))).toBe("plugin_outdated");
        expect(needsPlugin(missing)).toBe(true);
        expect(needsPlugin(outdated)).toBe(true);
    });

    test("ignores every other refusal", () => {
        expect(needsPlugin(rejection("INVALID", "no page", { details: { field: "pageId" } }))).toBe(false);
        expect(needsPlugin(rejection("EXTERNAL", "down", { details: { code: "plugin_missing" } }))).toBe(false);
        expect(needsPlugin(rejection("INVALID", "odd", { details: { code: "something_else" } }))).toBe(false);
        expect(needsPlugin(null)).toBe(false);
    });

    test("never toasts, because the refusal carries no field", () => {
        expect(react(rejection("INVALID", "the plugin is not installed", { details: { code: "plugin_missing" } })).kind).toBe(
            "form",
        );
    });
});
