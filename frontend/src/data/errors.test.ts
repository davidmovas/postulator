import { describe, expect, test } from "vitest";

import type { Code } from "../lib/errors.js";
import { copy } from "../copy/index.js";
import {
    browserSettingsPath,
    errorMessageOf,
    failure,
    fieldErrorOf,
    formErrorOf,
    isTorClosedToLinks,
    messages,
    needsPlugin,
    pluginCodeOf,
    providerMessageOf,
    react,
    validationErrorOf,
} from "./errors.js";

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

    test("a Tor Browser that takes no links says how to make it take them", () => {
        const thrown = rejection("CONFLICT", "Tor Browser is open but was started without taking links", {
            details: { code: "tor_closed_to_links" },
        });
        expect(react(thrown)).toEqual({ kind: "external", message: copy.app.torClosedToLinks });
        expect(isTorClosedToLinks(thrown)).toBe(true);
        expect(isTorClosedToLinks(rejection("CONFLICT"))).toBe(false);
    });

    test("a missing Tor Browser is a setup reaction that names where to fix it", () => {
        const thrown = rejection("INVALID", "Tor Browser is not installed", {
            details: { code: "tor_missing" },
        });
        expect(react(thrown)).toEqual({
            kind: "setup",
            message: copy.app.torMissing,
            action: { label: copy.app.torSetUp, to: browserSettingsPath },
        });
    });

    test("another INVALID detail code is still a form error", () => {
        expect(react(rejection("INVALID", "no plugin", { details: { code: "plugin_missing" } }))).toEqual({
            kind: "form",
            message: "no plugin",
        });
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

describe("the messages a screen shows", () => {
    const cancelled = Object.assign(new Error("cancelled"), { name: "CancelError" });
    const cases: readonly {
        name: string;
        thrown: unknown;
        field: string | null;
        form: string | null;
        validation: string | null;
        message: string | null;
    }[] = [
        { name: "nothing thrown", thrown: null, field: null, form: null, validation: null, message: null },
        { name: "nothing yet", thrown: undefined, field: null, form: null, validation: null, message: null },
        { name: "a cancelled call", thrown: cancelled, field: null, form: null, validation: null, message: null },
        {
            name: "a locked store",
            thrown: rejection("LOCKED"),
            field: null,
            form: null,
            validation: null,
            message: null,
        },
        {
            name: "a refusal of the named field",
            thrown: rejection("INVALID", "the path must start with a slash", { details: { field: "path" } }),
            field: "the path must start with a slash",
            form: null,
            validation: null,
            message: "the path must start with a slash",
        },
        {
            name: "a refusal of another field",
            thrown: rejection("INVALID", "the title is too long", { details: { field: "title" } }),
            field: null,
            form: null,
            validation: null,
            message: "the title is too long",
        },
        {
            name: "a refusal that names no field",
            thrown: rejection("INVALID", "the sheet has no usable rows"),
            field: null,
            form: "the sheet has no usable rows",
            validation: "the sheet has no usable rows",
            message: "the sheet has no usable rows",
        },
        {
            name: "a refusal with no words",
            thrown: rejection("INVALID"),
            field: null,
            form: messages.INVALID,
            validation: messages.INVALID,
            message: messages.INVALID,
        },
        {
            name: "a conflict",
            thrown: rejection("CONFLICT"),
            field: null,
            form: messages.CONFLICT,
            validation: null,
            message: messages.CONFLICT,
        },
        {
            name: "a failure outside the app",
            thrown: rejection("EXTERNAL"),
            field: null,
            form: messages.EXTERNAL,
            validation: null,
            message: messages.EXTERNAL,
        },
        {
            name: "an internal failure",
            thrown: rejection("INTERNAL", "panic"),
            field: null,
            form: messages.INTERNAL,
            validation: null,
            message: messages.INTERNAL,
        },
    ];

    for (const held of cases) {
        test(held.name, () => {
            expect(fieldErrorOf(held.thrown, "path")).toBe(held.field);
            expect(formErrorOf(held.thrown)).toBe(held.form);
            expect(validationErrorOf(held.thrown)).toBe(held.validation);
            expect(errorMessageOf(held.thrown)).toBe(held.message);
        });
    }
});

describe("providerMessageOf", () => {
    test("reads what the provider itself said, and nothing else", () => {
        const told = rejection("UNAUTHORIZED", "the key was refused", {
            details: { providerMessage: "Incorrect API key provided: sk-***." },
        });
        expect(providerMessageOf(failure(told))).toBe("Incorrect API key provided: sk-***.");
        expect(providerMessageOf(failure(rejection("UNAUTHORIZED", "the key was refused")))).toBeNull();
        expect(providerMessageOf(failure(rejection("UNAUTHORIZED", "x", { details: { providerMessage: "" } })))).toBeNull();
    });
});
