import type { Finding } from "../../../data/types.js";
import type { FindingOption, FindingOptionKind } from "../../../generated/vocab.js";
import { findingBulkOptions, findingOptionKinds, isOneOf } from "../../../generated/vocab.js";

export const bulkLimit = 1000;

const openStatus = "open";

const bulkKinds: ReadonlySet<FindingOptionKind> = new Set<FindingOptionKind>(["model", "proposal"]);

export interface BulkChoice {
    name: FindingOption;
    kind: FindingOptionKind;
}

export function bulkOptions(code: string, status: string, findings: readonly Finding[]): BulkChoice[] {
    if (code === "" || status !== openStatus) {
        return [];
    }
    const offered = new Map<string, FindingOptionKind>();
    for (const held of findings) {
        if (held.code !== code || held.status !== openStatus) {
            continue;
        }
        for (const option of held.options ?? []) {
            if (isOneOf(findingOptionKinds, option.kind) && bulkKinds.has(option.kind)) {
                offered.set(option.name, option.kind);
            }
        }
    }
    return findingBulkOptions.flatMap((name) => {
        const kind = offered.get(name);
        return kind === undefined ? [] : [{ name, kind }];
    });
}

export function batches(ids: readonly string[], size: number = bulkLimit): string[][] {
    const out: string[][] = [];
    for (let at = 0; at < ids.length; at += size) {
        out.push(ids.slice(at, at + size));
    }
    return out;
}
