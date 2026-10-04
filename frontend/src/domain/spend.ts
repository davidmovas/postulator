import type { SpendSlice } from "../data/types.js";

export interface Spent {
    calls: number;
    failed: number;
    input: number;
    cachedInput: number;
    output: number;
    reasoning: number;
    usd: number;
}

export interface PurposeRow extends Spent {
    purpose: string;
    share: number;
}

export interface ModelRow extends Spent {
    provider: string;
    model: string;
    tier: string;
}

export interface StepRow extends Spent {
    step: string;
    share: number;
    reasoningShare: number;
}

interface Group extends Spent {
    key: string;
    first: SpendSlice;
}

const wholeNumbers = new Intl.NumberFormat("en", { maximumFractionDigits: 0 });

export function count(value: number): string {
    return wholeNumbers.format(value);
}

export function share(part: number, whole: number): number {
    if (!(whole > 0) || !(part > 0)) {
        return 0;
    }
    return Math.min(part / whole, 1);
}

export function percent(fraction: number): string {
    if (!(fraction > 0)) {
        return "0%";
    }
    if (fraction >= 1) {
        return "100%";
    }
    const whole = Math.round(fraction * 100);
    if (whole === 0) {
        return "<1%";
    }
    return whole === 100 ? ">99%" : `${whole}%`;
}

function spentOf(group: Group): Spent {
    return {
        calls: group.calls,
        failed: group.failed,
        input: group.input,
        cachedInput: group.cachedInput,
        output: group.output,
        reasoning: group.reasoning,
        usd: group.usd,
    };
}

function grouped(slices: readonly SpendSlice[] | null, keyOf: (slice: SpendSlice) => string): Group[] {
    const held = new Map<string, Group>();
    for (const slice of slices ?? []) {
        const key = keyOf(slice);
        const group = held.get(key) ?? {
            key,
            first: slice,
            calls: 0,
            failed: 0,
            input: 0,
            cachedInput: 0,
            output: 0,
            reasoning: 0,
            usd: 0,
        };
        group.calls += slice.calls;
        group.failed += slice.failed;
        group.input += slice.input;
        group.cachedInput += slice.cachedInput;
        group.output += slice.output;
        group.reasoning += slice.reasoning;
        group.usd += slice.usd;
        held.set(key, group);
    }
    return [...held.values()].sort(
        (left, right) =>
            right.usd - left.usd ||
            right.calls + right.failed - (left.calls + left.failed) ||
            left.key.localeCompare(right.key),
    );
}

function spentIn(groups: readonly Group[]): number {
    return groups.reduce((sum, group) => sum + group.usd, 0);
}

export function purposeRows(slices: readonly SpendSlice[] | null): PurposeRow[] {
    const groups = grouped(slices, (slice) => slice.purpose);
    const total = spentIn(groups);
    return groups.map((group) => ({ ...spentOf(group), purpose: group.key, share: share(group.usd, total) }));
}

export function modelRows(slices: readonly SpendSlice[] | null): ModelRow[] {
    return grouped(slices, (slice) => `${slice.provider}/${slice.model}/${slice.tier}`).map((group) => ({
        ...spentOf(group),
        provider: group.first.provider,
        model: group.first.model,
        tier: group.first.tier,
    }));
}

export function stepRows(slices: readonly SpendSlice[] | null): StepRow[] {
    const groups = grouped(slices, (slice) => slice.step);
    const total = spentIn(groups);
    return groups.map((group) => ({
        ...spentOf(group),
        step: group.key,
        share: share(group.usd, total),
        reasoningShare: share(group.reasoning, group.output),
    }));
}
