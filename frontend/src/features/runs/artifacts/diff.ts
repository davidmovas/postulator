import { plainText } from "../../../domain/text.js";
import { listAt, numberAt, record, stringAt } from "./decode.js";

const htmlFields: ReadonlySet<string> = new Set(["product_short_description", "variation_description"]);

const excerptLength = 160;

export interface FieldChangeView {
    key: string;
    before: string;
    beforeHash: string;
    after: string;
    afterHash: string;
}

export interface BlockChangeView {
    kind: string;
    hash: string;
    raw: string;
    owner: string;
    change: string;
}

export interface PageEditView {
    pageId: string;
    path: string;
    body: string;
    fields: readonly FieldChangeView[];
    blocks: readonly BlockChangeView[];
    wpId: number | null;
}

export interface EditPlanView {
    pages: readonly PageEditView[];
    hash: string;
}

export interface RevertedFieldView {
    key: string;
    outcome: string;
    detail: string;
}

export function fieldText(key: string, value: string): string {
    return htmlFields.has(key.split(":")[0] ?? key) ? plainText(value) : value;
}

export function blockExcerpt(raw: string): string {
    const text = plainText(raw);
    return text.length <= excerptLength ? text : `${text.slice(0, excerptLength - 1).trimEnd()}…`;
}

export function fieldChangesAt(held: unknown, key: string): FieldChangeView[] {
    const out: FieldChangeView[] = [];
    for (const entry of listAt(held, key)) {
        const named = stringAt(entry, "key");
        if (named !== "") {
            out.push({
                key: named,
                before: stringAt(entry, "before"),
                beforeHash: stringAt(entry, "beforeHash"),
                after: stringAt(entry, "after"),
                afterHash: stringAt(entry, "afterHash"),
            });
        }
    }
    return out;
}

export function blockChangesAt(held: unknown, key: string): BlockChangeView[] {
    const out: BlockChangeView[] = [];
    for (const entry of listAt(held, key)) {
        if (record(entry) !== null) {
            out.push({
                kind: stringAt(entry, "kind"),
                hash: stringAt(entry, "hash"),
                raw: stringAt(entry, "raw"),
                owner: stringAt(entry, "owner"),
                change: stringAt(entry, "change"),
            });
        }
    }
    return out;
}

export function revertedFieldsAt(held: unknown, key: string): RevertedFieldView[] {
    const out: RevertedFieldView[] = [];
    for (const entry of listAt(held, key)) {
        const named = stringAt(entry, "key");
        if (named !== "") {
            out.push({ key: named, outcome: stringAt(entry, "outcome"), detail: stringAt(entry, "detail") });
        }
    }
    return out;
}

export function editPlanView(held: unknown): EditPlanView | null {
    if (record(held) === null) {
        return null;
    }
    const pages: PageEditView[] = [];
    for (const entry of listAt(held, "pages")) {
        if (record(entry) !== null) {
            pages.push({
                pageId: stringAt(entry, "pageId"),
                path: stringAt(entry, "path"),
                body: stringAt(entry, "body"),
                fields: fieldChangesAt(entry, "fields"),
                blocks: blockChangesAt(entry, "blocks"),
                wpId: numberAt(entry, "wpId"),
            });
        }
    }
    return { pages, hash: stringAt(held, "hash") };
}
