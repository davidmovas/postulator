import { copy } from "../../../../copy/index.js";
import { tokens, usd } from "../../../../domain/format.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { flag, line, listed, num, record, records, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe;

function fileName(path: string): string {
    const parts = path.split(/[\\/]/);
    return parts[parts.length - 1] ?? path;
}

function mappingLines(mapping: Args | null): Line[] {
    if (mapping === null) {
        return [];
    }
    const columns = records(mapping, "columns");
    const lines: Line[] = [line("list", said.imports.columns(columns.length))];
    for (const column of columns) {
        lines.push(line("field", said.imports.column(text(column, "field") ?? "", text(column, "column") ?? "")));
    }
    const options = record(mapping, "options");
    if (options !== null) {
        for (const [key, value] of Object.entries(options)) {
            if (typeof value === "string" && value !== "") {
                lines.push(line("rule", said.imports.option(key, value)));
            }
        }
    }
    return lines;
}

export const describeSync: Describer = (tool) => (tool === "sync_site" ? view(said.sync.site, [line("note", said.sync.body)]) : null);

export const describeImports: Describer = (tool, args) => {
    switch (tool) {
        case "imports_apply": {
            const path = text(args, "path") ?? "";
            const lines: Line[] = [line("file", said.imports.file(path)), ...mappingLines(record(args, "mapping"))];
            const saveAs = text(args, "saveMappingAs");
            if (saveAs !== null) {
                lines.push(line("note", said.imports.saveAs(saveAs)));
            }
            lines.push(line("warn", said.imports.applyBody, "danger"));
            return view(said.imports.apply(fileName(path)), lines);
        }
        case "imports_export":
            return view(said.imports.export(text(args, "path") ?? ""), []);
        case "imports_save_mapping":
            return view(said.imports.saveMapping(text(args, "name") ?? ""), mappingLines(args));
        case "imports_delete_mapping":
            return view([`${said.imports.deleteMapping} `, ref("mapping", text(args, "id") ?? "")], []);
        default:
            return null;
    }
};

function modelRef(args: Args): string {
    return `${text(args, "provider") ?? ""}/${text(args, "model") ?? ""}`;
}

export const describeModels: Describer = (tool, args) => {
    switch (tool) {
        case "models_upsert": {
            const lines: Line[] = [];
            const context = num(args, "contextTokens");
            if (context !== null) {
                lines.push(line("rule", said.models.context(tokens(context))));
            }
            const output = num(args, "maxOutputTokens");
            if (output !== null) {
                lines.push(line("rule", said.models.output(tokens(output))));
            }
            const input = num(args, "inputUsdPerM");
            const out = num(args, "outputUsdPerM");
            if (input !== null && out !== null) {
                lines.push(line("money", said.models.prices(usd(input), usd(out))));
            }
            const rpm = num(args, "rpm");
            const tpm = num(args, "tpm");
            if (rpm !== null && tpm !== null) {
                lines.push(line("rule", said.models.limits(rpm, tokens(tpm))));
            }
            const capabilities = [
                flag(args, "supportsStructured") === true ? said.models.structured : null,
                flag(args, "supportsImages") === true ? said.models.images : null,
                flag(args, "reasoning") === true ? said.models.reasoning : null,
            ].filter((held): held is string => held !== null);
            if (capabilities.length > 0) {
                lines.push(line("list", listed(capabilities)));
            }
            return view(said.models.upsert(modelRef(args)), lines);
        }
        case "models_disable":
            return view(said.models.disable(modelRef(args)), []);
        case "models_set_profile":
            return view(said.models.setProfile(text(args, "role") ?? "", modelRef(args)), []);
        case "models_set_provider_key":
            return view(said.models.setKey(text(args, "provider") ?? ""), [line("key", said.models.setKeyBody)]);
        case "models_test_provider":
            return view(said.models.test(modelRef(args)), [line("money", said.models.testBody)]);
        default:
            return null;
    }
};

export const describeContent: Describer = (tool, args) =>
    tool === "content_judge_page"
        ? view([`${said.content.judge} `, ref("page", text(args, "pageId") ?? "")], [line("money", said.content.body)])
        : null;

function scheduleLines(args: Args): Line[] {
    const lines: Line[] = [];
    const name = text(args, "name");
    if (name !== null) {
        lines.push(line("field", said.schedules.renamed(name)));
    }
    const cron = text(args, "cron");
    if (cron !== null) {
        lines.push(line("clock", said.schedules.cron(cron)));
    }
    const every = num(args, "intervalMinutes");
    if (every !== null && every > 0) {
        lines.push(line("clock", said.schedules.every(every)));
    }
    const entity = text(args, "entityId");
    if (entity !== null) {
        lines.push(line("target", [`${said.schedules.entity} `, ref("entity", entity)]));
    }
    const status = text(args, "status");
    if (status !== null) {
        lines.push(line("rule", said.schedules.status(status)));
    }
    const limit = num(args, "limit");
    if (limit !== null && limit > 0) {
        lines.push(line("rule", said.schedules.limit(limit)));
    }
    const template = text(args, "templateId");
    if (template !== null) {
        lines.push(line("target", [`${said.schedules.template} `, ref("template", template)]));
    }
    const steps = strings(args, "steps");
    if (steps.length > 0) {
        lines.push(line("step", said.runs.steps(listed(steps.map(stepLabel)))));
    }
    const publish = text(args, "publishMode");
    if (publish !== null) {
        lines.push(line(publish === "publish" ? "warn" : "note", publish === "publish" ? said.runs.publishes : said.runs.drafts, publish === "publish" ? "warn" : "muted"));
    }
    const maxUsd = num(args, "maxUsd");
    if (maxUsd !== null && maxUsd > 0) {
        lines.push(line("money", said.runs.budget(usd(maxUsd))));
    }
    const maxTokens = num(args, "maxTokens");
    if (maxTokens !== null && maxTokens > 0) {
        lines.push(line("money", said.runs.tokens(tokens(maxTokens))));
    }
    const enabled = flag(args, "enabled");
    if (enabled !== null) {
        lines.push(line("clock", enabled ? said.schedules.enabled : said.schedules.paused));
    }
    return lines;
}

export const describeSchedules: Describer = (tool, args) => {
    switch (tool) {
        case "schedules_create":
            return view(said.schedules.create(text(args, "name") ?? ""), scheduleLines(args).filter((held) => !(typeof held.parts[0] === "string" && held.parts[0].startsWith("Renamed"))));
        case "schedules_update":
            return view([`${said.schedules.update} `, ref("schedule", text(args, "id") ?? "")], scheduleLines(args));
        case "schedules_delete":
            return view([`${said.schedules.delete} `, ref("schedule", text(args, "id") ?? "")], []);
        case "schedules_enable":
            return view([`${said.schedules.enable} `, ref("schedule", text(args, "id") ?? "")], []);
        case "schedules_disable":
            return view([`${said.schedules.disable} `, ref("schedule", text(args, "id") ?? "")], []);
        case "schedules_run_now":
            return view([`${said.schedules.runNow} `, ref("schedule", text(args, "id") ?? "")], []);
        default:
            return null;
    }
};
