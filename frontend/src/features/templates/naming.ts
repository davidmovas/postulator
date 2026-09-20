import { copy } from "../../copy/index.js";

const maxAttempts = 200;

export function copyName(source: string, taken: readonly string[]): string {
    const used = new Set(taken.map((name) => name.trim().toLowerCase()));
    const first = copy.templates.copyName(source);
    if (!used.has(first.toLowerCase())) {
        return first;
    }
    for (let index = 2; index <= maxAttempts; index += 1) {
        const candidate = copy.templates.copyNameNumbered(source, index);
        if (!used.has(candidate.toLowerCase())) {
            return candidate;
        }
    }
    return copy.templates.copyNameNumbered(source, Date.now());
}
