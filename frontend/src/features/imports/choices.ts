import type { ImportChoice, PreviewPage } from "../../data/types.js";

export type Choices = ReadonlyMap<string, string>;

export const noChoices: Choices = new Map();

const ambiguous = "ambiguous";

export function withChoice(choices: Choices, rowKey: string, pageId: string): Choices {
    return new Map(choices).set(rowKey, pageId);
}

export function choicesOf(choices: Choices): ImportChoice[] {
    return [...choices]
        .sort(([left], [right]) => (left < right ? -1 : left > right ? 1 : 0))
        .map(([rowKey, pageId]) => ({ rowKey, pageId }));
}

function keyOf(page: PreviewPage): string {
    return page.rowKey ?? "";
}

export function pickable(page: PreviewPage): boolean {
    return keyOf(page) !== "" && (page.candidates ?? []).length > 0;
}

export function needsUrl(page: PreviewPage): boolean {
    return page.action === ambiguous && keyOf(page) === "";
}

export function chosenFor(choices: Choices, page: PreviewPage): string | null {
    const key = keyOf(page);
    return key === "" ? null : (choices.get(key) ?? null);
}
