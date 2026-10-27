import type { ProposedEntity, SiteMapNode } from "../../../data/types.js";

const rootPath = "/";

export function unmapped(node: SiteMapNode): boolean {
    return (node.entityId ?? "") === "" && node.page.path !== rootPath;
}

export function keywordLines(text: string): string[] {
    const seen = new Set<string>();
    const out: string[] = [];
    for (const raw of text.split(/\r?\n/)) {
        const line = raw.trim();
        const key = line.toLowerCase();
        if (line === "" || seen.has(key)) {
            continue;
        }
        seen.add(key);
        out.push(line);
    }
    return out;
}

export function joinsExisting(proposal: ProposedEntity): boolean {
    return (proposal.existingEntityId ?? "") !== "";
}

export interface Writes {
    written: number;
    joined: number;
}

export function writesOf(chosen: readonly ProposedEntity[]): Writes {
    const joined = chosen.filter(joinsExisting).length;
    return { written: chosen.length - joined, joined };
}
