import { copy } from "../../../../copy/index.js";
import { tokens as formatTokens } from "../../../../domain/format.js";

export const families = [
    "sites",
    "graph",
    "pages",
    "pagetypes",
    "templates",
    "runs",
    "sync",
    "reports",
    "imports",
    "findings",
    "models",
    "content",
    "schedules",
] as const;

export type Family = (typeof families)[number] | "other";

const nouns: Readonly<Record<string, [string, string]>> = {
    sites: ["site", "sites"],
    graph: ["entity", "entities"],
    pages: ["page", "pages"],
    pagetypes: ["page type", "page types"],
    templates: ["template", "templates"],
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

const plural: ReadonlySet<string> = new Set<string>(["list", "search"]);

function capitalised(phrase: string): string {
    return phrase === "" ? phrase : phrase[0].toUpperCase() + phrase.slice(1);
}

export function toolLabel(tool: string): string {
    const verb = verbOf(tool);
    const family = familyOf(tool);
    if (family === "other" || verb.includes(" ")) {
        return capitalised(verb);
    }
    const held = nouns[family];
    if (held === undefined) {
        return capitalised(verb);
    }
    return `${capitalised(verb)} ${plural.has(verb) ? held[1] : held[0]}`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

const refusalMark = "is not open to this conversation";

export function refusedText(text: string): boolean {
    return text.includes(refusalMark);
}

export interface Truncation {
    totalBytes: number;
    dropped: readonly { path: string; count: number }[];
    shortened: number;
    whole: boolean;
}

export function truncationOf(result: unknown): Truncation | null {
    if (!isRecord(result) || result["truncated"] !== true) {
        return null;
    }
    const counts = result["droppedItems"];
    const dropped: { path: string; count: number }[] = [];
    if (isRecord(counts)) {
        for (const [path, count] of Object.entries(counts)) {
            if (typeof count === "number" && count > 0) {
                dropped.push({ path, count });
            }
        }
        dropped.sort((first, second) => second.count - first.count);
    }
    return {
        totalBytes: typeof result["totalBytes"] === "number" ? result["totalBytes"] : 0,
        dropped,
        shortened: typeof result["shortenedText"] === "number" ? result["shortenedText"] : 0,
        whole: result["result"] !== undefined,
    };
}

export function kilobytes(bytes: number): number {
    return Math.max(1, Math.round(bytes / 1000));
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

function textAt(record: Record<string, unknown> | null, key: string): string {
    const held = record?.[key];
    return typeof held === "string" ? held : "";
}

function recordAt(record: Record<string, unknown> | null, key: string): Record<string, unknown> | null {
    const held = record?.[key];
    return isRecord(held) ? held : null;
}

function countAt(record: Record<string, unknown> | null, key: string): number {
    const held = record?.[key];
    return Array.isArray(held) ? held.length : 0;
}

function exploreSummary(result: Record<string, unknown>): string | null {
    const said = copy.agent.transcript.tool.explore;
    switch (result["view"]) {
        case "tree": {
            const tree = recordAt(result, "tree");
            const nodes = recordAt(tree, "nodes");
            return said.tree(countAt(nodes, "items"), textAt(recordAt(tree, "parent"), "path"), nodes?.["hasMore"] === true);
        }
        case "links": {
            const links = recordAt(result, "links");
            return said.links(textAt(recordAt(links, "page"), "path"), countAt(links, "required"));
        }
        case "gaps": {
            const gaps = recordAt(result, "gaps");
            return said.gaps(countAt(gaps, "items"), gaps?.["hasMore"] === true);
        }
        case "pack": {
            const pack = recordAt(result, "pack");
            const size = pack?.["tokens"];
            return said.pack(textAt(pack, "path"), formatTokens(typeof size === "number" ? size : 0));
        }
        default:
            return null;
    }
}

function cardsSummary(result: Record<string, unknown>): string | null {
    const said = copy.agent.transcript.tool.cards;
    const skipped = countAt(result, "skipped");
    const done = Array.isArray(result["entities"])
        ? said.written(countAt(result, "entities"))
        : Array.isArray(result["proposals"])
          ? said.proposed(countAt(result, "proposals"))
          : null;
    if (done === null) {
        return null;
    }
    return skipped === 0 ? done : `${done}, ${said.skipped(skipped)}`;
}

function pageSummary(result: Record<string, unknown>): string | null {
    const path = textAt(recordAt(recordAt(result, "page"), "page"), "path");
    return path === "" ? null : copy.agent.transcript.tool.explore.page(path);
}

const summaries: Readonly<Record<string, (result: Record<string, unknown>) => string | null>> = {
    pages_get: pageSummary,
    sites_explore: exploreSummary,
    graph_propose_cards: cardsSummary,
};

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
    const cut = truncationOf(result);
    if (cut !== null) {
        return copy.agent.transcript.tool.cutSummary(kilobytes(cut.totalBytes));
    }
    const special = summaries[tool]?.(result) ?? null;
    if (special !== null) {
        return special;
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
