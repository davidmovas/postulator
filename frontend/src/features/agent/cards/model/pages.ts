import { copy } from "../../../../copy/index.js";
import type { Keyword } from "../../../../data/types.js";
import { keywordsLine } from "../../../../domain/keywords.js";
import type { Args, Describer, Line } from "./card.js";
import { flag, has, line, num, opOf, record, records, ref, text, view } from "./card.js";
import { selectionLines } from "./selection.js";

const said = copy.agent.describe.pages;

const pagesManage = "pages_manage";
const pagesJudge = "pages_judge";

function keywordsOf(args: Args): Keyword[] {
    return records(args, "keywords").map((held) => {
        const volume = num(held, "volume");
        return volume === null ? { text: text(held, "text") ?? "" } : { text: text(held, "text") ?? "", volume };
    });
}

function pageFields(args: Args, withPath: boolean): Line[] {
    const lines: Line[] = [];
    const fields: readonly [string, (value: string) => string][] = [
        ["path", said.path],
        ["title", said.title],
        ["h1", said.h1],
        ["wpType", said.wpType],
        ["status", said.status],
        ["metaTitle", said.metaTitle],
        ["metaDescription", said.metaDescription],
    ];
    for (const [key, phrase] of fields) {
        const held = text(args, key);
        if (held !== null && (withPath || key !== "path")) {
            lines.push(line("field", phrase(held)));
        }
    }
    if (has(args, "keywords")) {
        const keywords = keywordsOf(args);
        lines.push(line("target", keywords.length === 0 ? said.keywordsOfEntity : said.keywords(keywordsLine(keywords))));
    }
    const entity = text(args, "entityId");
    if (entity !== null) {
        lines.push(line("target", [`${said.entity} `, ref("entity", entity)]));
    }
    const template = text(args, "templateId");
    if (template !== null) {
        lines.push(line("target", [`${said.template} `, ref("template", template)]));
    }
    const pageType = text(args, "pageTypeId");
    if (pageType !== null) {
        lines.push(line("target", [`${said.pageType} `, ref("pagetype", pageType)]));
    }
    return lines;
}

function page(args: Args) {
    return ref("page", text(args, "pageId") ?? "");
}

export const describePages: Describer = (tool, args) => {
    if (tool === pagesJudge) {
        return view([`${said.judge} `, page(args)], [line("money", said.judgeBody)]);
    }
    if (tool !== pagesManage) {
        return null;
    }
    const select = record(args, "select");
    const reset = flag(args, "reset") === true;
    switch (opOf(args)) {
        case "create":
            return view(said.create(text(args, "path") ?? ""), pageFields(args, false));
        case "update":
            return view([`${said.update} `, page(args)], pageFields(args, true));
        case "delete":
            return view(
                [`${said.delete} `, page(args)],
                [line("warn", flag(args, "onSite") === true ? said.deleteOnSite : said.deleteBody, "danger")],
            );
        case "map":
            return view([`${said.map} `, page(args), ` ${said.to} `, ref("entity", text(args, "entityId") ?? "")], []);
        case "unmap":
            return view([`${said.unmap} `, page(args)], []);
        case "canonical":
            return view(
                [`${said.canonical} `, page(args), ` ${said.canonicalOf} `, ref("entity", text(args, "entityId") ?? "")],
                [],
            );
        case "preview_link":
            return view([`${said.previewLink} `, page(args)], [line("link", said.previewLinkBody)]);
        case "set_types": {
            const type = text(args, "pageTypeId");
            const title =
                reset || type === null ? [said.resetTypes] : [`${said.setTypes} `, ref("pagetype", type)];
            return view(title, [...selectionLines(select), line("note", said.setManyBody)]);
        }
        case "set_templates": {
            const template = text(args, "templateId");
            const title =
                reset || template === null ? [said.resetTemplates] : [`${said.setTemplates} `, ref("template", template)];
            return view(title, [...selectionLines(select), line("note", said.setManyBody)]);
        }
        default:
            return null;
    }
};
