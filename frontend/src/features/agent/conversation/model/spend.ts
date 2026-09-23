export interface Spend {
    usd: number;
    calls: number;
    input: number;
    cachedInput: number;
}

export function cachedPercent(spend: Spend): number {
    if (spend.input <= 0 || spend.cachedInput <= 0) {
        return 0;
    }
    return Math.round((Math.min(spend.cachedInput, spend.input) / spend.input) * 100);
}
