export function spanMs(startedAt: string | null, finishedAt: string | null, now: number): number | null {
    if (startedAt === null || startedAt === "") {
        return null;
    }
    const from = Date.parse(startedAt);
    if (Number.isNaN(from)) {
        return null;
    }
    const to = finishedAt === null || finishedAt === "" ? now : Date.parse(finishedAt);
    if (Number.isNaN(to)) {
        return null;
    }
    return Math.max(to - from, 0);
}
