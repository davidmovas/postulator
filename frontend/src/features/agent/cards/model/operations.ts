import { copy } from "../../../../copy/index.js";
import { tokens, usd } from "../../../../domain/format.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { flag, line, listed, num, opOf, record, records, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe;

function fileName(path: string): string {
    const parts = path.split(/[\\/]/);
    return parts[parts.length - 1] ?? path;
}

function mappingLines(mapping: Args | null): Line[] {
    if (mapping === null) {
        return [];
    }
    const lines: Line[] = [];
    const saved = text(mapping, "id");
    const columns = records(mapping, "columns");
    if (saved !== null && columns.length === 0) {
        lines.push(line("rule", [`${said.imports.savedMapping} `, ref("mapping", saved)]));
    } else {
        lines.push(line("list", columns.length === 0 ? said.imports.detected : said.imports.columns(columns.length)));
    }
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

function sheetLines(args: Args): Line[] {
    return records(args, "sheets").map((sheet) => {
        const mapping = text(sheet, "mappingId");
        const named = said.imports.sheet(text(sheet, "sheet") ?? "", text(sheet, "rowType") ?? "pages");
        return mapping === null ? line("list", named) : line("list", [`${named} `, ref("mapping", mapping)]);
    });
}

export const describeImports: Describer = (tool, args) => {
    if (tool !== "imports") {
        return null;
    }
    switch (opOf(args)) {
        case "apply": {
            const path = text(args, "path") ?? "";
            const lines: Line[] = [line("file", said.imports.file(path)), ...mappingLines(record(args, "mapping")), ...sheetLines(args)];
            const saveAs = text(args, "saveMappingAs");
            if (saveAs !== null) {
                lines.push(line("note", said.imports.saveAs(saveAs)));
            }
            lines.push(line("warn", said.imports.applyBody, "danger"));
            return view(said.imports.apply(fileName(path)), lines);
        }
        case "export":
            return view(said.imports.export(text(args, "path") ?? ""), []);
        case "save_mapping": {
            const mapping = record(args, "mapping");
            return view(said.imports.saveMapping(mapping === null ? "" : (text(mapping, "name") ?? "")), mappingLines(mapping));
        }
        case "delete_mapping":
            return view([`${said.imports.deleteMapping} `, ref("mapping", text(args, "mappingId") ?? "")], []);
        default:
            return null;
    }
};

function scheduleLines(args: Args): Line[] {
    const lines: Line[] = [];
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
    const most = num(args, "maxPages");
    if (most !== null && most > 0) {
        lines.push(line("rule", said.schedules.limit(most)));
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
        const live = publish === "publish";
        lines.push(line(live ? "warn" : "note", live ? said.runs.publishes : said.runs.drafts, live ? "warn" : "muted"));
    }
    const maxUsd = num(args, "maxUsd");
    if (maxUsd !== null && maxUsd > 0) {
        lines.push(line("money", said.runs.budget(usd(maxUsd))));
    }
    const maxTokens = num(args, "maxTokens");
    if (maxTokens !== null && maxTokens > 0) {
        lines.push(line("money", said.runs.tokens(tokens(maxTokens))));
    }
    return lines;
}

function schedule(args: Args) {
    return ref("schedule", text(args, "scheduleId") ?? "");
}

export const describeSchedules: Describer = (tool, args) => {
    if (tool !== "schedules") {
        return null;
    }
    switch (opOf(args)) {
        case "create": {
            const lines = scheduleLines(args);
            const armed = flag(args, "enabled");
            if (armed !== null) {
                lines.push(line("clock", armed ? said.schedules.enabled : said.schedules.paused));
            }
            return view(said.schedules.create(text(args, "name") ?? ""), lines);
        }
        case "update": {
            const lines = scheduleLines(args);
            const name = text(args, "name");
            if (name !== null) {
                lines.unshift(line("field", said.schedules.renamed(name)));
            }
            return view([`${said.schedules.update} `, schedule(args)], lines);
        }
        case "delete":
            return view([`${said.schedules.delete} `, schedule(args)], []);
        case "enable":
            return view([`${said.schedules.enable} `, schedule(args)], []);
        case "disable":
            return view([`${said.schedules.disable} `, schedule(args)], []);
        case "run_now":
            return view([`${said.schedules.runNow} `, schedule(args)], []);
        default:
            return null;
    }
};
