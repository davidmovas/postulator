import { failure, messages } from "../../../data/errors.js";
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

export interface BulkOutcome {
    settled: number;
    total: number;
    refusal: unknown;
}

export async function settleInBatches(ids: readonly string[], settle: (batch: string[]) => Promise<unknown>): Promise<BulkOutcome> {
    let settled = 0;
    for (const batch of batches(ids)) {
        try {
            await settle(batch);
        } catch (refusal) {
            return { settled, total: ids.length, refusal };
        }
        settled += batch.length;
    }
    return { settled, total: ids.length, refusal: null };
}

export function refusalOf(thrown: unknown): string {
    const reported = failure(thrown);
    return reported.code === "INTERNAL" || reported.message === "" ? messages[reported.code] : reported.message;
}
