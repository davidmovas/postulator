export const families = [
    "sites",
    "graph",
    "pages",
    "templates",
    "policies",
    "runs",
    "sync",
    "reports",
    "imports",
    "models",
    "content",
    "schedules",
] as const;

export type Family = (typeof families)[number] | "other";

const nouns: Readonly<Record<string, [string, string]>> = {
    sites: ["site", "sites"],
    graph: ["entity", "entities"],
    pages: ["page", "pages"],
    templates: ["template", "templates"],
    policies: ["policy", "policies"],
    runs: ["run", "runs"],
    imports: ["mapping", "mappings"],
    models: ["model", "models"],
    schedules: ["schedule", "schedules"],
};

export function familyOf(tool: string): Family {
    const prefix = tool.split("_")[0] ?? "";
    return (families as readonly string[]).includes(prefix) ? (prefix as Family) : "other";
}

export function verbOf(tool: string): string {
    const parts = tool.split("_").filter((part) => part !== "");
    if (parts.length === 0) {
        return tool;
    }
    if (familyOf(tool) === "other") {
        return parts.join(" ");
    }
    return parts.slice(1).join(" ");
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function nounFor(tool: string, count: number): string {
    const held = nouns[familyOf(tool)];
    const [one, many] = held ?? ["item", "items"];
    if (tool.includes("edge")) {
        return count === 1 ? "edge" : "edges";
    }
    return count === 1 ? one : many;
}

function listSummary(tool: string, items: readonly unknown[], hasMore: boolean): string {
    if (items.length === 0) {
        return `no ${nounFor(tool, 0)}`;
    }
    const counted = `${items.length} ${nounFor(tool, items.length)}`;
    return hasMore ? `${counted}, more available` : counted;
}

const labelKeys = ["name", "path", "title", "label"] as const;

function labelOf(record: Record<string, unknown>): string | null {
    for (const key of labelKeys) {
        const held = record[key];
        if (typeof held === "string" && held !== "") {
            return held;
        }
    }
    return null;
}

export function resultSummary(tool: string, result: unknown): string {
    if (Array.isArray(result)) {
        return listSummary(tool, result, false);
    }
    if (typeof result === "string") {
        return result.length > 80 ? `${result.slice(0, 77)}…` : result;
    }
    if (typeof result === "number" || typeof result === "boolean") {
        return String(result);
    }
    if (!isRecord(result)) {
        return "done";
    }
    if (result["status"] === "confirmationRequired") {
        return "asked for approval";
    }
    if (result["truncated"] === true) {
        const bytes = typeof result["totalBytes"] === "number" ? result["totalBytes"] : 0;
        return `a long answer, ${Math.max(1, Math.round(bytes / 1000))} KB`;
    }
    if (Array.isArray(result["items"])) {
        return listSummary(tool, result["items"], result["hasMore"] === true);
    }
    if (typeof result["runId"] === "string") {
        return "run started";
    }
    const keys = Object.keys(result);
    if (keys.length === 0) {
        return "done";
    }
    if (keys.length === 1) {
        const only = result[keys[0] ?? ""];
        if (isRecord(only)) {
            const label = labelOf(only);
            if (label !== null) {
                return label;
            }
        }
        if (typeof only === "boolean") {
            return "done";
        }
    }
    const label = labelOf(result);
    if (label !== null) {
        return label;
    }
    return keys.length === 1 ? "1 field" : `${keys.length} fields`;
}
