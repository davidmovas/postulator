import { copy } from "../../../../copy/index.js";
import { tokens, usd } from "../../../../domain/format.js";
import { fieldLabel } from "../../../pages/model/labels.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Args, CardView, Describer, Line } from "./card.js";
import { clipped, flag, line, listed, num, opOf, record, records, ref, refs, strings, text, view } from "./card.js";
import { pagesShown, selectionLines } from "./selection.js";

const said = copy.agent.describe.runs;

const runsControl = "runs_control";

const briefLines = ["goal", "focus", "tone", "avoid"] as const;

function briefOf(brief: Args | null): Line[] {
    if (brief === null) {
        return [];
    }
    const lines: Line[] = [];
    for (const key of briefLines) {
        const held = text(brief, key);
        if (held !== null) {
            lines.push(line("note", said.brief[key](clipped(held))));
        }
    }
    const reference = text(brief, "reference");
    if (reference !== null) {
        lines.push(line("file", said.brief.reference([...reference].length)));
    }
    const style = text(brief, "styleSample");
    if (style !== null) {
        lines.push(line("file", said.brief.styleSample([...style].length)));
    }
    return lines;
}

function directiveLines(directives: readonly Args[]): Line[] {
    if (directives.length === 0) {
        return [];
    }
    const lines: Line[] = [line("list", said.directives(directives.length))];
    for (const directive of directives) {
        const parts: string[] = [];
        if (text(directive, "notes") !== null) {
            parts.push(said.directive.notes);
        }
        const sections = strings(directive, "addSections").length;
        if (sections > 0) {
            parts.push(said.directive.sections(sections));
        }
        const keywords = strings(directive, "keywords").length;
        if (keywords > 0) {
            parts.push(said.directive.keywords(keywords));
        }
        const links = records(directive, "mustLink").length;
        if (links > 0) {
            parts.push(said.directive.mustLink(links));
        }
        const least = num(directive, "minWords") ?? 0;
        const most = num(directive, "maxWords") ?? 0;
        if (least > 0 || most > 0) {
            parts.push(said.directive.words(least, most));
        }
        lines.push(line("target", [ref("page", text(directive, "pageId") ?? ""), `: ${listed(parts)}`]));
    }
    return lines;
}

function startView(args: Args, pricing: boolean): CardView {
    const pages = strings(args, "pageIds");
    const lines: Line[] = [];
    if (pages.length > 0) {
        lines.push(line("target", [`${said.pages} `, ...refs("page", pages, pagesShown)]));
    }
    lines.push(...selectionLines(record(args, "select")));
    const template = text(args, "templateId");
    lines.push(template === null ? line("target", said.templateDefault) : line("target", [`${said.template} `, ref("template", template)]));
    const steps = strings(args, "steps");
    lines.push(line("step", steps.length === 0 ? said.stepsDefault : said.steps(listed(steps.map(stepLabel)))));
    const fields = strings(args, "fields");
    if (fields.length > 0) {
        lines.push(line("field", said.fields(listed(fields.map(fieldLabel)))));
    }
    const rewrite = strings(args, "rewrite");
    if (rewrite.length > 0) {
        lines.push(line("warn", [`${said.rewrite} `, ...refs("page", rewrite, pagesShown)], "warn"));
    }
    const publish = text(args, "publishMode") === "publish";
    lines.push(line(publish ? "warn" : "note", publish ? said.publishes : said.drafts, publish ? "warn" : "muted"));
    const maxUsd = num(args, "maxUsd");
    lines.push(maxUsd !== null && maxUsd > 0 ? line("money", said.budget(usd(maxUsd))) : line("money", said.budgetDefault));
    const maxTokens = num(args, "maxTokens");
    if (maxTokens !== null && maxTokens > 0) {
        lines.push(line("money", said.tokens(tokens(maxTokens))));
    }
    lines.push(...briefOf(record(args, "brief")));
    lines.push(...directiveLines(records(args, "directives")));
    if (flag(args, "addEdges") === true) {
        lines.push(line("link", said.addEdges));
    }
    const kind = text(args, "kind") ?? "generate";
    const title = pages.length > 0 ? said.start(kind, pages.length) : said.startSelection(kind);
    return view(pricing ? said.estimate(title) : title, lines);
}

function run(args: Args) {
    return ref("run", text(args, "runId") ?? "");
}

export const describeRuns: Describer = (tool, args) => {
    if (tool !== runsControl) {
        return null;
    }
    switch (opOf(args)) {
        case "start":
            return startView(args, false);
        case "estimate":
            return startView(args, true);
        case "pause": {
            const reason = text(args, "reason");
            return view([`${said.pause} `, run(args)], reason === null ? [] : [line("note", said.reason(reason))]);
        }
        case "resume":
            return view([`${said.resume} `, run(args)], []);
        case "cancel":
            return view([`${said.cancel} `, run(args)], [line("note", said.cancelBody)]);
        case "retry_step":
            return view(
                [`${said.retry} `, ref("item", text(args, "itemId") ?? "")],
                flag(args, "acceptFindings") === true ? [line("warn", said.acceptFindings, "warn")] : [],
            );
        case "revert": {
            const lines: Line[] = [line("warn", said.revertBody, "warn")];
            const page = text(args, "pageId");
            if (page !== null) {
                lines.unshift(line("target", [`${said.revertPage} `, ref("page", page)]));
            }
            const fields = strings(args, "fields");
            if (fields.length > 0) {
                lines.unshift(line("field", said.revertFields(listed(fields.map(fieldLabel)))));
            }
            return view([`${said.revert} `, run(args)], lines);
        }
        case "regenerate":
            return view([`${said.regenerate(strings(args, "itemIds").length)} `, run(args)], [
                line("note", said.regenerateBody),
            ]);
        default:
            return null;
    }
};
