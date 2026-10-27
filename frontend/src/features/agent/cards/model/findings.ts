import { copy } from "../../../../copy/index.js";
import type { Finding } from "../../../../data/types.js";
import { optionDoes, optionLabel } from "../../../findings/labels.js";
import { findingCodeLabel } from "../../../pages/model/labels.js";
import { findingSentence } from "../../../findings/model/subject.js";
import type { Describer, Line, Part } from "./card.js";
import { line, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe.findings;

function titleOf(finding: Finding | null): Part[] {
    if (finding === null) {
        return [said.resolveAny];
    }
    const code = findingCodeLabel(finding.code);
    const page = finding.pageId ?? "";
    return page === "" ? [said.resolveSite(code)] : [said.resolve(code), ref("page", page)];
}

export const describeFindings: Describer = (tool, args, context) => {
    if (tool !== "findings") {
        return null;
    }
    const option = text(args, "option") ?? "";
    if (text(args, "op") === "resolve_many") {
        const code = findingCodeLabel(text(args, "code") ?? "");
        return view(said.resolveMany(code, strings(args, "findingIds").length), [
            line("rule", said.option(optionLabel(option), optionDoes(option)), "accent"),
        ]);
    }
    if (text(args, "op") !== "resolve") {
        return null;
    }
    const finding = context.finding;
    const lines: Line[] = [];
    if (finding !== null) {
        lines.push(line("note", findingSentence(finding.code, finding.subject)));
    }
    lines.push(line("rule", said.option(optionLabel(option), optionDoes(option)), "accent"));
    const pageId = text(args, "pageId");
    if (option === "map_to_page" && pageId !== null) {
        lines.push(line("target", [`${said.mapTo} `, ref("page", pageId)]));
    }
    const pageTypeId = text(args, "pageTypeId");
    if (option === "set_type" && pageTypeId !== null) {
        lines.push(line("target", [`${said.setType} `, ref("pagetype", pageTypeId)]));
    }
    return view(titleOf(finding), lines);
};
