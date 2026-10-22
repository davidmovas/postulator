import type { Finding } from "./decode.js";
import { boolAt, findingsAt, listAt, numberAt, record, stringAt, stringsAt } from "./decode.js";
import type { BlockChangeView, FieldChangeView, RevertedFieldView } from "./diff.js";
import { blockChangesAt, fieldChangesAt, revertedFieldsAt } from "./diff.js";

export interface ProductEditView {
    shortWritten: boolean;
    added: readonly string[];
    imageSet: boolean;
}

export interface PublishView {
    url: string;
    status: string;
    contentHash: string;
    wpId: number | null;
    created: boolean;
    seoApplied: readonly string[];
    skipped: readonly string[];
    findings: readonly Finding[];
    product: ProductEditView | null;
    recorded: boolean;
    fields: readonly FieldChangeView[];
    blocks: readonly BlockChangeView[];
    declined: readonly string[];
}

function productEditOf(held: unknown): ProductEditView | null {
    if (record(held) === null) {
        return null;
    }
    const image = numberAt(held, "imageId");
    return { shortWritten: boolAt(held, "shortWritten"), added: stringsAt(held, "added"), imageSet: image !== null && image !== 0 };
}

export function publishView(held: unknown): PublishView | null {
    const root = record(held);
    if (root === null) {
        return null;
    }
    return {
        url: stringAt(held, "url"),
        status: stringAt(held, "status"),
        contentHash: stringAt(held, "contentHash"),
        wpId: numberAt(held, "wpId"),
        created: boolAt(held, "created"),
        seoApplied: stringsAt(held, "seoApplied"),
        skipped: stringsAt(held, "skipped"),
        findings: findingsAt(held, "findings"),
        product: productEditOf(root["previousProduct"]),
        recorded: "fields" in root || "blocks" in root,
        fields: fieldChangesAt(held, "fields"),
        blocks: blockChangesAt(held, "blocks"),
        declined: stringsAt(held, "declined"),
    };
}

export interface RevertNeighbourView {
    pageId: string;
    path: string;
    outcome: string;
    detail: string;
}

export interface RevertView {
    path: string;
    outcome: string;
    detail: string;
    neighbours: readonly RevertNeighbourView[];
    fields: readonly RevertedFieldView[];
    findings: readonly Finding[];
}

export function revertView(held: unknown): RevertView | null {
    if (record(held) === null) {
        return null;
    }
    const neighbours: RevertNeighbourView[] = [];
    for (const entry of listAt(held, "neighbors")) {
        if (record(entry) !== null) {
            neighbours.push({
                pageId: stringAt(entry, "pageId"),
                path: stringAt(entry, "path"),
                outcome: stringAt(entry, "outcome"),
                detail: stringAt(entry, "detail"),
            });
        }
    }
    return {
        path: stringAt(held, "path"),
        outcome: stringAt(held, "outcome"),
        detail: stringAt(held, "detail"),
        neighbours,
        fields: revertedFieldsAt(held, "fields"),
        findings: findingsAt(held, "findings"),
    };
}
