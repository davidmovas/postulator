import { copy, errorMessages } from "../copy/index.js";
import type { Code, TransportError } from "../lib/errors.js";
import { parseError } from "../lib/errors.js";
import { isCancellation } from "./call.js";

export type { Code, TransportError } from "../lib/errors.js";

export const messages: Readonly<Record<Code, string>> = Object.freeze({ ...errorMessages });

export const codes: readonly Code[] = Object.freeze(Object.keys(messages) as Code[]);

const minimumRetryDelayMs = 250;
const defaultRetryDelayMs = 1000;

export const browserSettingsPath = "/settings/browser";

export const torMissingCode = "tor_missing";

export const torClosedToLinksCode = "tor_closed_to_links";

export interface ReactionAction {
    label: string;
    to: string;
}

export type Reaction =
    | { kind: "silent" }
    | { kind: "unlock" }
    | { kind: "field"; field: string; message: string }
    | { kind: "form"; message: string }
    | { kind: "throttle"; afterMs: number; message: string }
    | { kind: "refetch"; message: string }
    | { kind: "credentials"; message: string }
    | { kind: "budget"; message: string }
    | { kind: "review"; message: string }
    | { kind: "external"; message: string }
    | { kind: "setup"; message: string; action: ReactionAction }
    | { kind: "fatal"; message: string };

export function failure(thrown: unknown): TransportError {
    return parseError(thrown);
}

export function fieldOf(reported: TransportError): string | null {
    const held = reported.details?.["field"];
    return typeof held === "string" && held !== "" ? held : null;
}

export function retryAfterOf(reported: TransportError): number {
    const requested = reported.retry?.afterMs ?? defaultRetryDelayMs;
    return Math.max(requested, minimumRetryDelayMs);
}

export function react(thrown: unknown): Reaction {
    if (isCancellation(thrown)) {
        return { kind: "silent" };
    }
    const reported = parseError(thrown);
    switch (reported.code) {
        case "CANCELLED":
            return { kind: "silent" };
        case "LOCKED":
            return { kind: "unlock" };
        case "INVALID": {
            if (detailCodeOf(reported) === torMissingCode) {
                return {
                    kind: "setup",
                    message: copy.app.torMissing,
                    action: { label: copy.app.torSetUp, to: browserSettingsPath },
                };
            }
            const message = reported.message === "" ? messages.INVALID : reported.message;
            const field = fieldOf(reported);
            return field === null ? { kind: "form", message } : { kind: "field", field, message };
        }
        case "RATE_LIMITED":
            return { kind: "throttle", afterMs: retryAfterOf(reported), message: messages.RATE_LIMITED };
        case "NOT_FOUND":
            return { kind: "refetch", message: messages.NOT_FOUND };
        case "CONFLICT":
            if (detailCodeOf(reported) === torClosedToLinksCode) {
                return { kind: "external", message: copy.app.torClosedToLinks };
            }
            return { kind: "refetch", message: messages.CONFLICT };
        case "UNAUTHORIZED":
            return { kind: "credentials", message: messages.UNAUTHORIZED };
        case "BUDGET_EXCEEDED":
            return { kind: "budget", message: messages.BUDGET_EXCEEDED };
        case "NEEDS_HUMAN":
            return { kind: "review", message: messages.NEEDS_HUMAN };
        case "EXTERNAL":
            return { kind: "external", message: messages.EXTERNAL };
        case "INTERNAL":
            return { kind: "fatal", message: messages.INTERNAL };
        default:
            return { kind: "fatal", message: messages.INTERNAL };
    }
}

const pluginCodes: ReadonlySet<string> = new Set(["plugin_missing", "plugin_outdated"]);

export function providerMessageOf(reported: TransportError): string | null {
    const held = reported.details?.["providerMessage"];
    return typeof held === "string" && held !== "" ? held : null;
}

export function detailCodeOf(reported: TransportError): string | null {
    const held = reported.details?.["code"];
    return typeof held === "string" && held !== "" ? held : null;
}

export function isTorClosedToLinks(thrown: unknown): boolean {
    if (thrown === null || thrown === undefined || isCancellation(thrown)) {
        return false;
    }
    const reported = parseError(thrown);
    return reported.code === "CONFLICT" && detailCodeOf(reported) === torClosedToLinksCode;
}

export function pluginCodeOf(reported: TransportError): string | null {
    if (reported.code !== "INVALID") {
        return null;
    }
    const held = detailCodeOf(reported);
    return held !== null && pluginCodes.has(held) ? held : null;
}

export function needsPlugin(thrown: unknown): boolean {
    if (thrown === null || thrown === undefined) {
        return false;
    }
    return pluginCodeOf(parseError(thrown)) !== null;
}

export function fieldErrorOf(thrown: unknown, field: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "field" && reaction.field === field ? reaction.message : null;
}

export function formErrorOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    switch (reaction.kind) {
        case "silent":
        case "unlock":
        case "field":
            return null;
        default:
            return reaction.message;
    }
}

export function keyErrorOf(thrown: unknown, key: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reported = failure(thrown);
    if (reported.code !== "INVALID") {
        return null;
    }
    const held = reported.details?.["key"];
    return held === key ? (reported.message === "" ? messages.INVALID : reported.message) : null;
}
