import type { TemplateSpec } from "../../../../data/types.js";

export type RefKind = "site" | "entity" | "page" | "template" | "policy" | "schedule" | "run" | "edge" | "mapping" | "item" | "override";

export interface Ref {
    kind: RefKind;
    id: string;
}

export type Part = string | Ref;

export type LineIcon = "target" | "field" | "list" | "money" | "warn" | "link" | "key" | "file" | "clock" | "step" | "rule" | "note";

export type LineTone = "muted" | "info" | "warn" | "danger" | "ok" | "accent";

export interface Line {
    icon: LineIcon;
    tone: LineTone;
    parts: readonly Part[];
}

export interface CardView {
    title: readonly Part[];
    lines: readonly Line[];
    generic: boolean;
}

export interface DescribeContext {
    currentTemplate: TemplateSpec | null;
}

export type Args = Readonly<Record<string, unknown>>;

export type Describer = (tool: string, args: Args, context: DescribeContext) => CardView | null;

export function ref(kind: RefKind, id: string): Ref {
    return { kind, id };
}

export function line(icon: LineIcon, parts: readonly Part[] | string, tone: LineTone = "muted"): Line {
    return { icon, tone, parts: typeof parts === "string" ? [parts] : parts };
}

export function view(title: readonly Part[] | string, lines: readonly Line[]): CardView {
    return { title: typeof title === "string" ? [title] : title, lines, generic: false };
}

export function isArgs(value: unknown): value is Args {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function has(args: Args, key: string): boolean {
    return args[key] !== undefined && args[key] !== null;
}

export function text(args: Args, key: string): string | null {
    const held = args[key];
    return typeof held === "string" && held !== "" ? held : null;
}

export function flag(args: Args, key: string): boolean | null {
    const held = args[key];
    return typeof held === "boolean" ? held : null;
}

export function num(args: Args, key: string): number | null {
    const held = args[key];
    return typeof held === "number" && Number.isFinite(held) ? held : null;
}

export function strings(args: Args, key: string): string[] {
    const held = args[key];
    return Array.isArray(held) ? held.filter((item): item is string => typeof item === "string") : [];
}

export function records(args: Args, key: string): Args[] {
    const held = args[key];
    return Array.isArray(held) ? held.filter(isArgs) : [];
}

export function record(args: Args, key: string): Args | null {
    const held = args[key];
    return isArgs(held) ? held : null;
}

export function masked(value: string): boolean {
    return value.includes("***");
}

export function joined(items: readonly string[]): string {
    if (items.length <= 1) {
        return items.join("");
    }
    return `${items.slice(0, -1).join(", ")} and ${items[items.length - 1] ?? ""}`;
}

export function listed(items: readonly string[]): string {
    return items.join(", ");
}
