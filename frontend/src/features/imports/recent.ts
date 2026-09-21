export interface RecentFile {
    path: string;
    rows: number;
    at: string;
}

export const recentCap = 5;

export function fileName(path: string): string {
    const cut = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
    const name = cut < 0 ? path : path.slice(cut + 1);
    return name === "" ? path : name;
}

export function remember(held: readonly RecentFile[], entry: RecentFile): RecentFile[] {
    if (entry.path === "") {
        return [...held];
    }
    return [entry, ...held.filter((row) => row.path !== entry.path)].slice(0, recentCap);
}

export function forget(held: readonly RecentFile[], path: string): RecentFile[] {
    return held.filter((row) => row.path !== path);
}

function keyOf(siteId: string): string {
    return `postulator.import.recent.${siteId}`;
}

function sane(held: unknown): RecentFile[] {
    if (!Array.isArray(held)) {
        return [];
    }
    const out: RecentFile[] = [];
    for (const row of held as readonly unknown[]) {
        if (typeof row !== "object" || row === null) {
            continue;
        }
        const entry = row as Record<string, unknown>;
        if (typeof entry["path"] !== "string" || entry["path"] === "") {
            continue;
        }
        out.push({
            path: entry["path"],
            rows: typeof entry["rows"] === "number" ? entry["rows"] : 0,
            at: typeof entry["at"] === "string" ? entry["at"] : "",
        });
    }
    return out.slice(0, recentCap);
}

export function readRecent(siteId: string): RecentFile[] {
    if (siteId === "") {
        return [];
    }
    try {
        const held = window.localStorage.getItem(keyOf(siteId));
        return held === null ? [] : sane(JSON.parse(held));
    } catch {
        return [];
    }
}

export function writeRecent(siteId: string, held: readonly RecentFile[]): void {
    if (siteId === "") {
        return;
    }
    try {
        window.localStorage.setItem(keyOf(siteId), JSON.stringify(held.slice(0, recentCap)));
    } catch {
        return;
    }
}
