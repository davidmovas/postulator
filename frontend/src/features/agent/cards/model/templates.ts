import { copy } from "../../../../copy/index.js";
import type { TemplateSpec } from "../../../../data/types.js";
import type { JsonObject, JsonValue } from "../../../../domain/merge-patch.js";
import { mergePatch } from "../../../../domain/merge-patch.js";
import { sentencesOf } from "../../../templates/sentences.js";
import { draftFromJson, isJsonObject, jsonOf } from "../../../templates/spec.js";
import type { Args, Describer, Line } from "./card.js";
import { line, object, ref, text, view } from "./card.js";

const said = copy.agent.describe.templates;

function profilesOf(listed: JsonValue): JsonValue {
    if (!Array.isArray(listed)) {
        return listed;
    }
    const profiles: JsonObject = {};
    for (const held of listed) {
        if (!isJsonObject(held) || typeof held.role !== "string") {
            continue;
        }
        profiles[held.role] = { provider: held.provider ?? "", model: held.model ?? "" };
    }
    return profiles;
}

function recipeOf(listed: JsonValue): JsonValue {
    if (!Array.isArray(listed)) {
        return listed;
    }
    return listed.map((held) => {
        if (!isJsonObject(held)) {
            return held;
        }
        const { allowErrors, iterations, ...rest } = held;
        const params: JsonObject = {};
        if (allowErrors !== undefined) {
            params.allowErrors = allowErrors;
        }
        if (iterations !== undefined) {
            params.iterations = iterations;
        }
        return Object.keys(params).length === 0 ? rest : { ...rest, params };
    });
}

function parsed(held: Args | null): JsonObject | null {
    if (held === null) {
        return null;
    }
    const decoded: JsonValue = held as JsonValue;
    if (!isJsonObject(decoded)) {
        return null;
    }

    const spec: JsonObject = { ...decoded };
    if (spec.modelProfiles !== undefined) {
        spec.modelProfiles = profilesOf(spec.modelProfiles);
    }
    if (spec.recipe !== undefined) {
        spec.recipe = recipeOf(spec.recipe);
    }
    return spec;
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
            const spec = parsed(object(args, "spec"));
            lines.push(...(spec === null ? [line("warn", said.unreadableSpec, "warn")] : specLines(spec)));
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "templates_update": {
            const lines = identityLines(args);
            const raw = object(args, "spec");
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
            const patch = parsed(object(args, "patch"));
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
