export const minimumDockWidth = 280;
export const maximumDockWidth = 640;
export const defaultDockWidth = 340;

export const globalDockKey = "global";

export function dockKey(siteId: string | null): string {
    return siteId === null || siteId === "" ? globalDockKey : siteId;
}

export function clampWidth(width: number): number {
    return Math.min(Math.max(Math.round(width), minimumDockWidth), maximumDockWidth);
}

export function parseWidth(raw: string | null): number {
    const parsed = raw === null ? Number.NaN : Number.parseInt(raw, 10);
    return Number.isNaN(parsed) ? defaultDockWidth : clampWidth(parsed);
}

export function chooseConversation(
    remembered: string | undefined,
    available: readonly { id: string }[],
): string | null {
    if (remembered !== undefined && available.some((held) => held.id === remembered)) {
        return remembered;
    }
    return available[0]?.id ?? null;
}

export function parseChoices(raw: string | null): Record<string, string> {
    if (raw === null) {
        return {};
    }
    let decoded: unknown;
    try {
        decoded = JSON.parse(raw);
    } catch {
        return {};
    }
    if (typeof decoded !== "object" || decoded === null || Array.isArray(decoded)) {
        return {};
    }
    const out: Record<string, string> = {};
    for (const [key, value] of Object.entries(decoded)) {
        if (typeof value === "string") {
            out[key] = value;
        }
    }
    return out;
}
