import type { Spend } from "../../agent/conversation/model/spend.js";
import { cachedPercent } from "../../agent/conversation/model/spend.js";

export interface UsageSummary {
    usd: number;
    calls: number;
    usage: {
        input: number;
        cachedInput: number;
        total: number;
    };
}

function spendOf(summary: UsageSummary): Spend {
    return {
        usd: summary.usd,
        calls: summary.calls,
        input: summary.usage.input,
        cachedInput: summary.usage.cachedInput,
    };
}

export function cachedShare(summary: UsageSummary): number | null {
    const share = cachedPercent(spendOf(summary));
    return share === 0 ? null : share;
}
