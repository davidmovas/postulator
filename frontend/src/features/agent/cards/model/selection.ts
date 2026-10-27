import { copy } from "../../../../copy/index.js";
import type { Args, Line } from "./card.js";
import { line, ref, refs, strings, text } from "./card.js";

const said = copy.agent.describe.selection;

export const pagesShown = 6;

export function selectionLines(select: Args | null): Line[] {
    if (select === null) {
        return [];
    }
    const lines: Line[] = [];
    const pages = strings(select, "pageIds");
    if (pages.length > 0) {
        lines.push(line("target", [`${said.pages} `, ...refs("page", pages, pagesShown)]));
    }
    const under = strings(select, "under");
    if (under.length > 0) {
        lines.push(line("target", [`${said.under} `, ...refs("page", under, pagesShown)]));
    }
    const ofType = text(select, "ofType");
    if (ofType !== null) {
        lines.push(line("target", [`${under.length > 0 ? said.ofType : said.everyOfType} `, ref("pagetype", ofType)]));
    }
    const except = strings(select, "except");
    if (except.length > 0) {
        lines.push(line("target", [`${said.except} `, ...refs("page", except, pagesShown)]));
    }
    return lines;
}
