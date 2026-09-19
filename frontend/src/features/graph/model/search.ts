import type { Entity } from "../../../data/types.js";
import { compareNames } from "./index.js";

export type SearchField = "name" | "primaryKeyword" | "secondaryKeyword";

export interface SearchHit {
    id: string;
    score: number;
    field: SearchField;
}

function wordStart(haystack: string, at: number): boolean {
    return at === 0 || !/[\p{L}\p{N}]/u.test(haystack[at - 1]);
}

function strength(haystack: string, needle: string): number {
    const at = haystack.indexOf(needle);
    if (at < 0) {
        return 0;
    }
    if (at === 0) {
        return 3;
    }
    return wordStart(haystack, at) ? 2 : 1;
}

const weights: Readonly<Record<SearchField, readonly [number, number, number]>> = {
    name: [4, 3, 2],
    primaryKeyword: [1.5, 1.4, 1.2],
    secondaryKeyword: [1, 0.9, 0.8],
};

function scoreOf(field: SearchField, haystack: string, needle: string): number {
    const found = strength(haystack.toLowerCase(), needle);
    return found === 0 ? 0 : weights[field][3 - found];
}

export function rank(entities: readonly Entity[], query: string, limit: number): SearchHit[] {
    const needle = query.trim().toLowerCase();
    if (needle === "") {
        return [];
    }
    const hits: SearchHit[] = [];
    for (const held of entities) {
        let best: SearchHit | null = null;
        const consider = (field: SearchField, text: string): void => {
            const score = scoreOf(field, text, needle);
            if (score > 0 && (best === null || score > best.score)) {
                best = { id: held.id, score, field };
            }
        };
        consider("name", held.name);
        consider("primaryKeyword", held.primaryKeyword);
        for (const keyword of held.secondaryKeywords ?? []) {
            consider("secondaryKeyword", keyword);
        }
        if (best !== null) {
            hits.push(best);
        }
    }
    const names = new Map(entities.map((held) => [held.id, held.name]));
    hits.sort((left, right) => {
        if (left.score !== right.score) {
            return right.score - left.score;
        }
        const named = compareNames(names.get(left.id) ?? "", names.get(right.id) ?? "");
        return named !== 0 ? named : compareNames(left.id, right.id);
    });
    return hits.slice(0, limit);
}
