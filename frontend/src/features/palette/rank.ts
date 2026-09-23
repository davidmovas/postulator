export function scoreOf(query: string, text: string): number {
    if (query === "") {
        return 3;
    }
    const needle = query.toLowerCase();
    const haystack = text.toLowerCase();
    if (haystack.startsWith(needle)) {
        return 0;
    }
    const at = haystack.indexOf(needle);
    if (at < 0) {
        return -1;
    }
    return /[\s/\-_.]/.test(haystack[at - 1] ?? "") ? 1 : 2;
}

export function rankBy<T>(query: string, items: readonly T[], text: (item: T) => string, limit: number): T[] {
    const scored: { item: T; score: number; at: number }[] = [];
    items.forEach((item, at) => {
        const score = scoreOf(query, text(item));
        if (score >= 0) {
            scored.push({ item, score, at });
        }
    });
    scored.sort((left, right) => (left.score === right.score ? left.at - right.at : left.score - right.score));
    return scored.slice(0, limit).map((held) => held.item);
}
