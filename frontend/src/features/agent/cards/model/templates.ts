import { copy } from "../../../../copy/index.js";
import type { TemplateSpec } from "../../../../data/types.js";
import type { JsonObject, JsonValue } from "../../../../domain/merge-patch.js";
import { mergePatch } from "../../../../domain/merge-patch.js";
import { sentencesOf } from "../../../templates/sentences.js";
import { draftFromJson, isJsonObject, jsonOf } from "../../../templates/spec.js";
import type { Args, Describer, Line } from "./card.js";
import { line, ref, text, view } from "./card.js";

const said = copy.agent.describe.templates;

function parsed(raw: string | null): JsonObject | null {
    if (raw === null) {
        return null;
    }
    try {
        const decoded: JsonValue = JSON.parse(raw);
        return isJsonObject(decoded) ? decoded : null;
    } catch {
        return null;
    }
}

function specLines(spec: JsonObject): Line[] {
    return sentencesOf(draftFromJson({}), spec).map((sentence) => line("rule", sentence));
}

function changeLines(current: TemplateSpec, next: JsonObject): Line[] {
    const base = jsonOf(current);
    const patch = mergePatch(base, next);
    if (patch === undefined) {
        return [line("note", said.nothingChanges)];
    }
    if (!isJsonObject(patch)) {
        return specLines(next);
    }
    return sentencesOf(draftFromJson(base), patch).map((sentence) => line("rule", sentence));
}

function identityLines(args: Args): Line[] {
    const lines: Line[] = [];
    const name = text(args, "name");
    if (name !== null) {
        lines.push(line("field", said.renamed(name)));
    }
    const kind = text(args, "pageKind");
    if (kind !== null) {
        lines.push(line("field", said.pageKind(kind)));
    }
    return lines;
}

export const describeTemplates: Describer = (tool, args, context) => {
    switch (tool) {
        case "templates_create": {
            const lines: Line[] = [];
            const kind = text(args, "pageKind");
            if (kind !== null) {
                lines.push(line("field", said.pageKind(kind)));
            }
            const site = text(args, "siteId");
            lines.push(text(args, "scope") === "site" && site !== null ? line("target", [`${said.scopeSite} `, ref("site", site)]) : line("target", said.scopeGlobal));
            const spec = parsed(text(args, "spec"));
            lines.push(...(spec === null ? [line("warn", said.unreadableSpec, "warn")] : specLines(spec)));
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "templates_update": {
            const lines = identityLines(args);
            const raw = text(args, "spec");
            if (raw !== null) {
                const spec = parsed(raw);
                if (spec === null) {
                    lines.push(line("warn", said.unreadableSpec, "warn"));
                } else if (context.currentTemplate !== null) {
                    lines.push(...changeLines(context.currentTemplate, spec));
                } else {
                    lines.push(line("note", said.replaces), ...specLines(spec));
                }
            }
            return view([`${said.update} `, ref("template", text(args, "id") ?? "")], lines);
        }
        case "templates_delete":
            return view([`${said.delete} `, ref("template", text(args, "id") ?? "")], [line("warn", said.deleteBody, "danger")]);
        case "templates_set_override": {
            const patch = parsed(text(args, "patch"));
            const base = context.currentTemplate === null ? draftFromJson({}) : draftFromJson(jsonOf(context.currentTemplate));
            const lines = patch === null ? [line("warn", said.unreadableSpec, "warn")] : sentencesOf(base, patch).map((sentence) => line("rule", sentence));
            const page = text(args, "scope") === "page";
            return view(
                [
                    `${said.override} `,
                    ref("template", text(args, "templateId") ?? ""),
                    ` ${page ? said.forPage : said.forSite} `,
                    ref(page ? "page" : "site", text(args, "targetId") ?? ""),
                ],
                lines,
            );
        }
        case "templates_delete_override":
            return view([`${said.deleteOverride} `, ref("override", text(args, "id") ?? "")], []);
        default:
            return null;
    }
};
