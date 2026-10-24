import type { Finding } from "../../../data/types.js";
import type { FindingOption } from "../../../generated/vocab.js";
import { findingBulkOptions } from "../../../generated/vocab.js";

export const bulkLimit = 1000;

const openStatus = "open";
const modelKind = "model";

export function bulkOptions(code: string, status: string, findings: readonly Finding[]): FindingOption[] {
    if (code === "" || status !== openStatus) {
        return [];
    }
    const offered = new Set<string>();
    for (const held of findings) {
        if (held.code !== code || held.status !== openStatus) {
            continue;
        }
        for (const option of held.options ?? []) {
            if (option.kind === modelKind) {
                offered.add(option.name);
            }
        }
    }
    return findingBulkOptions.filter((option) => offered.has(option));
}

export function batches(ids: readonly string[], size: number = bulkLimit): string[][] {
    const out: string[][] = [];
    for (let at = 0; at < ids.length; at += size) {
        out.push(ids.slice(at, at + size));
    }
    return out;
}
