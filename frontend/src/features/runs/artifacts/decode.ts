export type Severity = "error" | "warn" | "info";

export const severityError: Severity = "error";
export const severityWarn: Severity = "warn";

export type LinkClass = "graph" | "self" | "external" | "unknown_internal";

export const codeTargetMissing = "target_missing";

export const linkClassByCode: Readonly<Record<string, LinkClass>> = {
    self_link: "self",
    external_link: "external",
    unknown_internal_link: "unknown_internal",
};

export const linkClasses: readonly LinkClass[] = ["graph", "self", "external", "unknown_internal"];

export function decodeArtifact(content: string): unknown {
    if (content === "") {
        return null;
    }
    try {
        return JSON.parse(content) as unknown;
    } catch {
        return null;
    }
}

export function record(held: unknown): Record<string, unknown> | null {
    if (typeof held !== "object" || held === null || Array.isArray(held)) {
        return null;
    }
    return held as Record<string, unknown>;
}

export function stringAt(held: unknown, key: string): string {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return typeof value === "string" ? value : "";
}

export function numberAt(held: unknown, key: string): number | null {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return typeof value === "number" && Number.isFinite(value) ? value : null;
}

export function boolAt(held: unknown, key: string): boolean {
    const source = record(held);
    return source !== null && source[key] === true;
}

export function listAt(held: unknown, key: string): readonly unknown[] {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return Array.isArray(value) ? (value as readonly unknown[]) : [];
}

export function stringsAt(held: unknown, key: string): readonly string[] {
    const out: string[] = [];
    for (const entry of listAt(held, key)) {
        if (typeof entry === "string" && entry !== "") {
            out.push(entry);
        }
    }
    return out;
}

export function fieldAt(held: unknown, key: string): unknown {
    const source = record(held);
    return source === null ? null : source[key];
}

export interface Finding {
    severity: Severity;
    code: string;
    message: string;
    details: Record<string, unknown> | null;
}

export function severityOf(held: unknown): Severity {
    const raw = stringAt(held, "severity");
    if (raw === severityError || raw === severityWarn) {
        return raw;
    }
    return "info";
}

export function findingsAt(held: unknown, key: string): readonly Finding[] {
    const out: Finding[] = [];
    for (const entry of listAt(held, key)) {
        if (record(entry) === null) {
            continue;
        }
        out.push({
            severity: severityOf(entry),
            code: stringAt(entry, "code"),
            message: stringAt(entry, "message"),
            details: record(fieldAt(entry, "details")),
        });
    }
    return out;
}

export function weigh(findings: readonly Finding[]): { errors: number; warnings: number } {
    let errors = 0;
    let warnings = 0;
    for (const finding of findings) {
        if (finding.severity === severityError) {
            errors += 1;
        } else if (finding.severity === severityWarn) {
            warnings += 1;
        }
    }
    return { errors, warnings };
}
