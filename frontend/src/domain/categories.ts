import { copy } from "../copy/index.js";
import type { Entity, Page } from "../data/types.js";
import type { CategoryTrailItem } from "../ui/category-trail.js";

export type Category = NonNullable<Page["categories"]>[number];

export interface FiledTerm {
    entityId: string;
    name: string;
    termId: number;
    created: boolean;
}

export type CategoryStanding = { state: "off" } | { state: "onSite"; termId: number } | { state: "onPublish" };

type Filed = Pick<Entity, "id" | "siteCategory" | "categories">;

function termOf(category: Category): number | null {
    const held = category.termId;
    return typeof held === "number" && Number.isFinite(held) && held > 0 ? held : null;
}

function standingItem(category: Category): CategoryTrailItem {
    const termId = termOf(category);
    return termId === null
        ? { key: category.entityId, name: category.name, state: "onPublish", hint: copy.categories.onPublish }
        : { key: category.entityId, name: category.name, state: "onSite", hint: copy.categories.onSite(termId) };
}

export function pageCategoryItems(page: Pick<Page, "categories" | "categoriesNeedPlugin">): readonly CategoryTrailItem[] {
    const chain = page.categories ?? [];
    if (!page.categoriesNeedPlugin) {
        return chain.map(standingItem);
    }
    return chain.map((category) => ({
        key: category.entityId,
        name: category.name,
        state: "needsPlugin",
        hint: copy.categories.needsPlugin,
    }));
}

export function entityCategoryItems(entity: Pick<Entity, "categories">): readonly CategoryTrailItem[] {
    return (entity.categories ?? []).map(standingItem);
}

export function filedItems(terms: readonly FiledTerm[]): readonly CategoryTrailItem[] {
    return terms.map((term) => ({
        key: term.entityId,
        name: term.name,
        state: "onSite",
        hint: term.created ? copy.categories.createdByRun(term.termId) : copy.categories.onSite(term.termId),
    }));
}

export function becomesCategory(): CategoryTrailItem {
    return { key: "becomes", name: copy.categories.becomes, state: "becomes", hint: copy.categories.becomesHint };
}

export function categoryStanding(entity: Filed): CategoryStanding {
    if (!entity.siteCategory) {
        return { state: "off" };
    }
    const own = (entity.categories ?? []).find((category) => category.entityId === entity.id);
    const termId = own === undefined ? null : termOf(own);
    return termId === null ? { state: "onPublish" } : { state: "onSite", termId };
}

export function ownCategoryItem(entity: Filed, name: string): CategoryTrailItem | null {
    const standing = categoryStanding(entity);
    switch (standing.state) {
        case "off":
            return null;
        case "onSite":
            return { key: entity.id, name, state: "onSite", hint: copy.categories.onSite(standing.termId) };
        case "onPublish":
            return { key: entity.id, name, state: "onPublish", hint: copy.categories.onPublish };
    }
}
