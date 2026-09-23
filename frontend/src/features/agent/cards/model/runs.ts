import { copy } from "../../../../copy/index.js";
import { tokens, usd } from "../../../../domain/format.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Describer, Line, Part } from "./card.js";
import { line, listed, num, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe.runs;

const maxPagesShown = 6;

export const describeRuns: Describer = (tool, args) => {
    switch (tool) {
        case "runs_start": {
            const pages = strings(args, "pageIds");
            const lines: Line[] = [];
            const shown: Part[] = [`${said.pages} `];
            pages.slice(0, maxPagesShown).forEach((id, index) => {
                if (index > 0) {
                    shown.push(", ");
                }
                shown.push(ref("page", id));
            });
            if (pages.length > maxPagesShown) {
                shown.push(` ${copy.agent.describe.more(pages.length - maxPagesShown)}`);
            }
            lines.push(line("target", shown));
            const template = text(args, "templateId");
            lines.push(template === null ? line("target", said.templateDefault) : line("target", [`${said.template} `, ref("template", template)]));
            const steps = strings(args, "steps");
            lines.push(line("step", steps.length === 0 ? said.stepsDefault : said.steps(listed(steps.map(stepLabel)))));
            const publish = text(args, "publishMode") === "publish";
            lines.push(line(publish ? "warn" : "note", publish ? said.publishes : said.drafts, publish ? "warn" : "muted"));
            const maxUsd = num(args, "maxUsd");
            const maxTokens = num(args, "maxTokens");
            if (maxUsd !== null && maxUsd > 0) {
                lines.push(line("money", said.budget(usd(maxUsd))));
            } else {
                lines.push(line("money", said.noBudget, "warn"));
            }
            if (maxTokens !== null && maxTokens > 0) {
                lines.push(line("money", said.tokens(tokens(maxTokens))));
            }
            return view(said.start(text(args, "kind") ?? "generate", pages.length), lines);
        }
        case "runs_pause": {
            const reason = text(args, "reason");
            return view([`${said.pause} `, ref("run", text(args, "runId") ?? "")], reason === null ? [] : [line("note", said.reason(reason))]);
        }
        case "runs_resume":
            return view([`${said.resume} `, ref("run", text(args, "runId") ?? "")], []);
        case "runs_cancel":
            return view([`${said.cancel} `, ref("run", text(args, "runId") ?? "")], [line("note", said.cancelBody)]);
        case "runs_retry_step":
            return view([`${said.retry} `, ref("item", text(args, "itemId") ?? "")], []);
        case "runs_regenerate":
            return view([`${said.regenerate(strings(args, "itemIds").length)} `, ref("run", text(args, "runId") ?? "")], [
                line("note", said.regenerateBody),
            ]);
        default:
            return null;
    }
};
