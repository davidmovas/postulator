export type Code =
    | "NOT_FOUND"
    | "CONFLICT"
    | "INVALID"
    | "UNAUTHORIZED"
    | "RATE_LIMITED"
    | "BUDGET_EXCEEDED"
    | "EXTERNAL"
    | "INTERNAL"
    | "CANCELLED"
    | "NEEDS_HUMAN"
    | "LOCKED";

export interface Retry {
    afterMs: number;
}

export interface TransportError {
    code: Code;
    message: string;
    details?: Record<string, unknown>;
    retry?: Retry;
}

const codes: ReadonlySet<string> = new Set<string>([
    "NOT_FOUND",
    "CONFLICT",
    "INVALID",
    "UNAUTHORIZED",
    "RATE_LIMITED",
    "BUDGET_EXCEEDED",
    "EXTERNAL",
    "INTERNAL",
    "CANCELLED",
    "NEEDS_HUMAN",
    "LOCKED",
]);

const fallback: TransportError = { code: "INTERNAL", message: "unexpected internal error" };

export function parseError(thrown: unknown): TransportError {
    if (typeof thrown !== "object" || thrown === null || !("cause" in thrown)) {
        return fallback;
    }

    const cause = (thrown as { cause: unknown }).cause;
    if (typeof cause !== "object" || cause === null) {
        return fallback;
    }

    const candidate = cause as Partial<TransportError>;
    if (typeof candidate.code !== "string" || !codes.has(candidate.code)) {
        return fallback;
    }
    if (typeof candidate.message !== "string") {
        return fallback;
    }

    const parsed: TransportError = { code: candidate.code, message: candidate.message };
    if (candidate.details !== undefined) {
        parsed.details = candidate.details;
    }
    if (candidate.retry !== undefined && typeof candidate.retry.afterMs === "number") {
        parsed.retry = { afterMs: candidate.retry.afterMs };
    }
    return parsed;
}

export function isCode(thrown: unknown, code: Code): boolean {
    return parseError(thrown).code === code;
}
