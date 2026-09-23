import { copy } from "../../../../copy/index.js";
import type { ToolSchema } from "../../../../data/types.js";
import { verbOf } from "../../conversation/model/tools.js";
import type { Args, CardView, DescribeContext, Describer, Line } from "./card.js";
import { isArgs, line, listed, masked } from "./card.js";
import { describeGraph } from "./graph.js";
import { describeContent, describeImports, describeModels, describeSchedules, describeSync } from "./operations.js";
import { describePages } from "./pages.js";
import { describePolicies } from "./policies.js";
import { describeRuns } from "./runs.js";
import { describeSites } from "./sites.js";
import { describeTemplates } from "./templates.js";

const describers: readonly Describer[] = [
    describeSites,
    describeGraph,
    describePages,
    describeTemplates,
    describePolicies,
    describeRuns,
    describeSync,
    describeImports,
    describeModels,
    describeContent,
    describeSchedules,
];

const maxTextShown = 240;

function humanKey(key: string): string {
    return key.replace(/([a-z0-9])([A-Z])/g, "$1 $2").replace(/_/g, " ").toLowerCase();
}

function labelFor(key: string, schema: ToolSchema | null): string {
    const held = schema?.properties?.[key]?.description;
    return held === undefined || held === "" ? humanKey(key) : held;
}

function scalar(value: unknown): string | null {
    if (typeof value === "string") {
        if (masked(value)) {
            return copy.agent.describe.hidden;
        }
        return value.length > maxTextShown ? `${value.slice(0, maxTextShown - 1)}…` : value;
    }
    if (typeof value === "number") {
        return String(value);
    }
    if (typeof value === "boolean") {
        return value ? copy.agent.describe.yes : copy.agent.describe.no;
    }
    return null;
}

function genericLines(args: Args, schema: ToolSchema | null, depth: number): Line[] {
    const lines: Line[] = [];
    const known = Object.keys(schema?.properties ?? {});
    const keys = [...known.filter((key) => key in args), ...Object.keys(args).filter((key) => !known.includes(key))];
    for (const key of keys) {
        const value = args[key];
        if (value === null || value === undefined) {
            continue;
        }
        const label = labelFor(key, schema);
        const indent = " ".repeat(depth);
        const plain = scalar(value);
        if (plain !== null) {
            lines.push(line("field", `${indent}${copy.agent.describe.field(label, plain)}`));
        } else if (Array.isArray(value)) {
            const items = value.map(scalar).filter((held): held is string => held !== null);
            if (items.length === value.length) {
                lines.push(line("list", `${indent}${copy.agent.describe.field(label, listed(items))}`));
            } else {
                lines.push(line("list", `${indent}${label}`));
                value.filter(isArgs).forEach((held) => {
                    lines.push(...genericLines(held, schema?.properties?.[key]?.items ?? null, depth + 1));
                });
            }
        } else if (isArgs(value)) {
            lines.push(line("list", `${indent}${label}`));
            lines.push(...genericLines(value, schema?.properties?.[key] ?? null, depth + 1));
        }
    }
    return lines;
}

export function describeAction(tool: string, args: unknown, schema: ToolSchema | null, context: DescribeContext): CardView {
    const held: Args = isArgs(args) ? args : {};
    for (const describer of describers) {
        const described = describer(tool, held, context);
        if (described !== null) {
            return described;
        }
    }
    return { title: [verbOf(tool)], lines: genericLines(held, schema, 0), generic: true };
}
