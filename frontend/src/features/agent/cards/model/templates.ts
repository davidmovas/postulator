import { copy } from "../../../../copy/index.js";
import type { TemplateSpec } from "../../../../data/types.js";
import type { JsonObject, JsonValue } from "../../../../domain/merge-patch.js";
import { mergePatch } from "../../../../domain/merge-patch.js";
import { sentencesOf } from "../../../templates/sentences.js";
import { draftFromJson, isJsonObject, jsonOf } from "../../../templates/spec.js";
import type { Args, Describer, Line } from "./card.js";
import { line, object, opOf, ref, text, view } from "./card.js";

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
    return lines;
}

function template(args: Args) {
    return ref("template", text(args, "templateId") ?? "");
}

export const describeTemplates: Describer = (tool, args, context) => {
    if (tool !== "templates") {
        return null;
    }
    switch (opOf(args)) {
        case "create": {
            const lines: Line[] = [line("target", text(args, "scope") === "site" ? said.scopeSite : said.scopeGlobal)];
            const spec = parsed(object(args, "spec"));
            lines.push(...(spec === null ? [line("warn", said.unreadableSpec, "warn")] : specLines(spec)));
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "update": {
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
            return view([`${said.update} `, template(args)], lines);
        }
        case "delete":
            return view([`${said.delete} `, template(args)], [line("warn", said.deleteBody, "danger")]);
        case "override": {
            const patch = parsed(object(args, "spec"));
            const base = context.currentTemplate === null ? draftFromJson({}) : draftFromJson(jsonOf(context.currentTemplate));
            const lines =
                patch === null
                    ? [line("warn", said.unreadableSpec, "warn")]
                    : sentencesOf(base, patch).map((sentence) => line("rule", sentence));
            const page = text(args, "pageId");
            return view(
                page === null
                    ? [`${said.override} `, template(args), ` ${said.forSite}`]
                    : [`${said.override} `, template(args), ` ${said.forPage} `, ref("page", page)],
                lines,
            );
        }
        case "delete_override":
            return view([`${said.deleteOverride} `, template(args)], [line("note", said.deleteOverrideBody)]);
        default:
            return null;
    }
};
