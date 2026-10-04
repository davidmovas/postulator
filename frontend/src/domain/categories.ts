import { copy } from "../copy/index.js";
import type { Page } from "../data/types.js";
import type { CategoryTrailItem } from "../ui/category-trail.js";

export type Category = NonNullable<Page["categories"]>[number];

export interface FiledTerm {
    name: string;
    termId: number;
    created: boolean;
}

function termOf(category: Category): number | null {
    const held = category.termId;
    return typeof held === "number" && Number.isFinite(held) && held > 0 ? held : null;
}

function standingItem(category: Category): CategoryTrailItem {
    const termId = termOf(category);
    return termId === null
        ? { key: category.id, name: category.name, state: "onPublish", hint: copy.categories.onPublish }
        : { key: category.id, name: category.name, state: "onSite", hint: copy.categories.onSite(termId) };
}

export function chainItems(chain: readonly Category[] | null | undefined): readonly CategoryTrailItem[] {
    return (chain ?? []).map(standingItem);
}

export function pageCategoryItems(page: Pick<Page, "categories" | "categoriesNeedPlugin">): readonly CategoryTrailItem[] {
    const chain = page.categories ?? [];
    if (!page.categoriesNeedPlugin) {
        return chain.map(standingItem);
    }
    return chain.map((category) => ({
        key: category.id,
        name: category.name,
        state: "needsPlugin",
        hint: copy.categories.needsPlugin,
    }));
}

export function filedItems(terms: readonly FiledTerm[]): readonly CategoryTrailItem[] {
    return terms.map((term) => ({
        key: String(term.termId),
        name: term.name,
        state: "onSite",
        hint: term.created ? copy.categories.createdByRun(term.termId) : copy.categories.onSite(term.termId),
    }));
}
