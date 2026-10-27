import { copy } from "../../../../copy/index.js";
import { tokens as formatTokens } from "../../../../domain/format.js";

export const families = [
    "sites",
    "graph",
    "pages",
    "links",
    "pagetypes",
    "templates",
    "runs",
    "imports",
    "findings",
    "models",
    "schedules",
    "plans",
] as const;

export type Family = (typeof families)[number] | "other";

const defaultActs: Readonly<Record<string, string>> = {
    findings: "list",
    plans: "list",
    pages_get: "model",
};

const nouns: Readonly<Record<string, [string, string]>> = {
    sites: ["site", "sites"],
    graph: ["entity", "entities"],
    pages: ["page", "pages"],
    links: ["link", "links"],
    pagetypes: ["page type", "page types"],
    templates: ["template", "templates"],
    runs: ["run", "runs"],
    imports: ["mapping", "mappings"],
    findings: ["finding", "findings"],
    models: ["model", "models"],
    schedules: ["schedule", "schedules"],
    plans: ["plan", "plans"],
};

const keyNouns: Readonly<Record<string, [string, string]>> = {
    sites: ["site", "sites"],
    runs: ["run", "runs"],
    items: ["page", "pages"],
    entities: ["entity", "entities"],
    edges: ["edge", "edges"],
    pages: ["page", "pages"],
    links: ["link", "links"],
    owed: ["link owed", "links owed"],
    inbound: ["link in", "links in"],
    suggestions: ["suggestion", "suggestions"],
    plans: ["plan", "plans"],
    rules: ["rule", "rules"],
    mappings: ["mapping", "mappings"],
    types: ["page type", "page types"],
    templates: ["template", "templates"],
    schedules: ["schedule", "schedules"],
    models: ["model", "models"],
    findings: ["finding", "findings"],
};

export function familyOf(tool: string): Family {
    const prefix = tool.split("_")[0] ?? "";
    return (families as readonly string[]).includes(prefix) ? (prefix as Family) : "other";
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function words(name: string): string {
    const spaced = name.split("_").filter((part) => part !== "").join(" ");
    return spaced === "" ? spaced : spaced[0].toUpperCase() + spaced.slice(1);
}

export function actOf(tool: string, args: unknown): string {
    if (isRecord(args)) {
        for (const key of ["op", "view"] as const) {
            const held = args[key];
            if (typeof held === "string" && held !== "") {
                return held;
            }
        }
    }
    return defaultActs[tool] ?? "";
}

const dangerous = "dangerous";
const write = "write";

const plainWrites: Readonly<Record<string, ReadonlySet<string>>> = {
    sites_manage: new Set(["create", "update", "sync"]),
    pages_manage: new Set(["create", "update", "map", "unmap", "canonical", "preview_link", "set_types", "set_templates"]),
    graph_edit: new Set([
        "create",
        "update",
        "anchors",
        "add_edge",
        "approve_edge",
        "reject_edge",
        "move",
        "rescore",
        "adopt_parents",
        "apply_proposals",
    ]),
    pagetypes: new Set(["save", "scope"]),
    templates: new Set(["create", "update", "override"]),
    runs_control: new Set(["start", "pause", "resume", "cancel", "retry_step", "regenerate"]),
    imports: new Set(["export", "save_mapping"]),
    schedules: new Set(["create", "update", "enable", "disable", "run_now"]),
};

export function riskOf(tool: string, args: unknown, base: string): string {
    if (base !== dangerous) {
        return base;
    }
    return plainWrites[tool]?.has(actOf(tool, args)) === true ? write : dangerous;
}

export function toolLabel(tool: string, args?: unknown): string {
    const known = copy.agent.acts[tool];
    if (known === undefined) {
        return words(tool);
    }
    const act = args === undefined ? "" : actOf(tool, args);
    return known.acts[act] ?? known.label;
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

function counted(noun: readonly [string, string], count: number): string {
    return `${count} ${count === 1 ? noun[0] : noun[1]}`;
}

function listSummary(noun: readonly [string, string], items: readonly unknown[], hasMore: boolean): string {
    if (items.length === 0) {
        return `no ${noun[1]}`;
    }
    const said = counted(noun, items.length);
    return hasMore ? `${said}, more available` : said;
}

function nounOf(tool: string): readonly [string, string] {
    if (tool.includes("edge")) {
        return ["edge", "edges"];
    }
    return nouns[familyOf(tool)] ?? ["item", "items"];
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

function listed(record: Record<string, unknown> | null): { items: readonly unknown[]; more: boolean } | null {
    const items = record?.["items"];
    return Array.isArray(items) ? { items, more: record?.["hasMore"] === true } : null;
}

function treeSummary(result: Record<string, unknown>): string | null {
    const nodes = listed(recordAt(result, "nodes"));
    if (nodes === null) {
        return null;
    }
    const under = textAt(recordAt(recordAt(result, "node"), "page"), "path");
    return copy.agent.transcript.tool.explore.tree(nodes.items.length, under, nodes.more);
}

function pageSummary(result: Record<string, unknown>): string | null {
    const said = copy.agent.transcript.tool.explore;
    const pack = recordAt(result, "pack");
    if (pack !== null) {
        const size = pack["tokens"];
        return said.pack(textAt(pack, "path"), formatTokens(typeof size === "number" ? size : 0));
    }
    const path = textAt(recordAt(recordAt(result, "page"), "page"), "path");
    if (path !== "") {
        return said.page(path);
    }
    return recordAt(result, "reports") === null ? null : said.reports;
}

function linksSummary(result: Record<string, unknown>): string | null {
    const said = copy.agent.transcript.tool.explore;
    const owed = listed(recordAt(result, "owed"));
    if (owed !== null) {
        return said.owed(owed.items.length, owed.more);
    }
    const suggestions = listed(recordAt(result, "suggestions"));
    if (suggestions !== null) {
        return said.suggestions(suggestions.items.length, suggestions.more);
    }
    const page = textAt(recordAt(result, "page"), "path");
    if (page !== "") {
        return said.links(page);
    }
    const root = textAt(recordAt(result, "root"), "path");
    return said.section(root);
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

function adoptionSummary(result: Record<string, unknown>): string | null {
    const adoption = recordAt(result, "adoption");
    if (adoption === null) {
        return null;
    }
    const adopting = adoption["adopting"];
    return copy.agent.transcript.tool.adoption(typeof adopting === "number" ? adopting : 0);
}

function sitesSummary(result: Record<string, unknown>): string | null {
    return recordAt(result, "overview") === null ? null : copy.agent.transcript.tool.overview;
}

function runsSummary(result: Record<string, unknown>): string | null {
    const said = copy.agent.transcript.tool.runs;
    const run = recordAt(result, "run");
    if (run !== null) {
        return said.run(textAt(run, "kind"), textAt(run, "status"));
    }
    const events = recordAt(result, "events");
    if (events !== null) {
        return said.events(countAt(events, "events"));
    }
    const artifact = recordAt(result, "artifact");
    if (artifact !== null) {
        return said.artifact;
    }
    return recordAt(result, "report") === null ? null : said.report;
}

const summaries: Readonly<Record<string, (result: Record<string, unknown>) => string | null>> = {
    sites: sitesSummary,
    pages_get: pageSummary,
    pages_tree: treeSummary,
    links: linksSummary,
    graph_propose: cardsSummary,
    graph_read: adoptionSummary,
    runs: runsSummary,
};

function wrappedSummary(tool: string, result: Record<string, unknown>): string | null {
    for (const [key, value] of Object.entries(result)) {
        const noun = key === "list" ? nounOf(tool) : keyNouns[key];
        const list = listed(isRecord(value) ? value : null);
        if (noun !== undefined && list !== null) {
            return listSummary(noun, list.items, list.more);
        }
        if (noun !== undefined && Array.isArray(value)) {
            return listSummary(noun, value, false);
        }
    }
    return null;
}

export function resultSummary(tool: string, result: unknown): string {
    if (Array.isArray(result)) {
        return listSummary(nounOf(tool), result, false);
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
        return listSummary(nounOf(tool), result["items"], result["hasMore"] === true);
    }
    if (typeof result["runId"] === "string") {
        return "run started";
    }
    const wrapped = wrappedSummary(tool, result);
    if (wrapped !== null) {
        return wrapped;
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
