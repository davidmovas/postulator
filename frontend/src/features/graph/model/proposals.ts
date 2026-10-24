import type { Page, ProposedEntity } from "../../../data/types.js";

const rootPath = "/";

export function unmapped(page: Page): boolean {
    return (page.entityId === null || page.entityId === "") && page.path !== rootPath;
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

export function placeOf(proposal: ProposedEntity): string {
    if (proposal.path !== undefined && proposal.path !== "") {
        return proposal.path;
    }
    return proposal.parent ?? "";
}

export function tickable(proposal: ProposedEntity): boolean {
    return (proposal.clash ?? "") === "";
}

export function joinsExisting(proposal: ProposedEntity): boolean {
    return (proposal.existingEntityId ?? "") !== "";
}

export function everyTickable(proposals: readonly ProposedEntity[]): ReadonlySet<number> {
    const out = new Set<number>();
    proposals.forEach((proposal, at) => {
        if (tickable(proposal)) {
            out.add(at);
        }
    });
    return out;
}

export function toggled(proposals: readonly ProposedEntity[], kept: ReadonlySet<number>, at: number): ReadonlySet<number> {
    const proposal = proposals[at];
    if (proposal === undefined || !tickable(proposal)) {
        return kept;
    }
    const next = new Set(kept);
    if (next.has(at)) {
        next.delete(at);
    } else {
        next.add(at);
    }
    return next;
}

export function chosenOf(proposals: readonly ProposedEntity[], kept: ReadonlySet<number>): ProposedEntity[] {
    return proposals.filter((proposal, at) => kept.has(at) && tickable(proposal));
}

export interface Writes {
    written: number;
    joined: number;
}

export function writesOf(chosen: readonly ProposedEntity[]): Writes {
    const joined = chosen.filter(joinsExisting).length;
    return { written: chosen.length - joined, joined };
}
