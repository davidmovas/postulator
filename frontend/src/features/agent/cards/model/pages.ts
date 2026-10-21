import { copy } from "../../../../copy/index.js";
import type { Args, Describer, Line } from "./card.js";
import { line, ref, text, view } from "./card.js";

const said = copy.agent.describe.pages;

function pageFields(args: Args): Line[] {
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
        if (held !== null) {
            lines.push(line("field", phrase(held)));
        }
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

export const describePages: Describer = (tool, args) => {
    switch (tool) {
        case "pages_create": {
            const lines = pageFields(args).filter((held) => !(typeof held.parts[0] === "string" && held.parts[0].startsWith("Path ")));
            return view(said.create(text(args, "path") ?? ""), lines);
        }
        case "pages_update":
            return view([`${said.update} `, ref("page", text(args, "id") ?? "")], pageFields(args));
        case "pages_delete":
            return view([`${said.delete} `, ref("page", text(args, "id") ?? "")], [line("warn", said.deleteBody, "danger")]);
        case "pages_map_to_entity":
            return view(
                [`${said.map} `, ref("page", text(args, "pageId") ?? ""), ` ${said.to} `, ref("entity", text(args, "entityId") ?? "")],
                [],
            );
        case "pages_unmap":
            return view([`${said.unmap} `, ref("page", text(args, "pageId") ?? "")], []);
        case "pages_set_canonical":
            return view(
                [
                    `${said.canonical} `,
                    ref("page", text(args, "pageId") ?? ""),
                    ` ${said.canonicalOf} `,
                    ref("entity", text(args, "entityId") ?? ""),
                ],
                [],
            );
        default:
            return null;
    }
};
