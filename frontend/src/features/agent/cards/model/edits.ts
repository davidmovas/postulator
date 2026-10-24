import { copy } from "../../../../copy/index.js";
import { plainText } from "../../../../domain/text.js";
import { fieldLabel } from "../../../pages/model/labels.js";
import { fieldText } from "../../../runs/artifacts/diff.js";
import type { Args, Describer, Line, Part } from "./card.js";
import { clipped, line, num, ref, text, view } from "./card.js";

const said = copy.agent.describe;

const startOfBody = -1;

export function pagePart(args: Args): Part {
    const id = text(args, "pageId");
    return id === null ? (text(args, "path") ?? "") : ref("page", id);
}

function newText(args: Args): Line {
    const shown = plainText(text(args, "html") ?? "");
    return shown === "" ? line("field", said.blocks.noText) : line("field", said.blocks.text(clipped(shown)));
}

export const describeFields: Describer = (tool, args) => {
    if (tool !== "pages_fields" || text(args, "op") !== "set") {
        return null;
    }
    const key = text(args, "key") ?? "";
    const value = fieldText(key, text(args, "value") ?? "");
    const lines: Line[] = [
        value === "" ? line("warn", said.fields.clears, "warn") : line("field", said.fields.value(clipped(value))),
        line("note", said.fields.run),
    ];
    return view([said.fields.set(fieldLabel(key)), pagePart(args)], lines);
};

export const describeBlocks: Describer = (tool, args) => {
    if (tool !== "pages_blocks") {
        return null;
    }
    const index = num(args, "index") ?? 0;
    const page = pagePart(args);
    const run = line("note", said.blocks.run);
    switch (text(args, "op")) {
        case "replace":
            return view([said.blocks.replace(index), page], [newText(args), run]);
        case "insert_after":
            return view(
                [index === startOfBody ? said.blocks.insertStart : said.blocks.insertAfter(index), page],
                [newText(args), run],
            );
        case "delete":
            return view([said.blocks.delete(index), page], [line("warn", said.blocks.deleteBody, "danger"), run]);
        case "place_link":
            return view(
                [said.blocks.placeLink(text(args, "anchor") ?? "", index), page],
                [line("link", [`${said.blocks.linkTo} `, ref("page", text(args, "targetId") ?? "")]), line("note", said.blocks.linkBody), run],
            );
        default:
            return null;
    }
};
