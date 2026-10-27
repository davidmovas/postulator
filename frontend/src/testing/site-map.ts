import type { SiteMapFilter, SiteMapNode, SiteMapView } from "../data/types.js";

interface Answer {
    view: SiteMapView;
    more: boolean;
}

export interface SiteMapFake {
    answers: Map<string, Answer>;
    asked: SiteMapFilter[];
    more: SiteMapFilter[];
    stable: Map<string, unknown>;
}

export const siteMapFake: SiteMapFake = { answers: new Map(), asked: [], more: [], stable: new Map() };

function keyOf(filter: SiteMapFilter): string {
    return JSON.stringify({
        parentId: filter.parentId ?? "",
        search: filter.search ?? "",
        pageTypeId: filter.pageTypeId ?? "",
        within: filter.within ?? "",
    });
}

export function resetSiteMap(): void {
    siteMapFake.answers.clear();
    siteMapFake.stable.clear();
    siteMapFake.asked.length = 0;
    siteMapFake.more.length = 0;
}

export function answerSiteMap(filter: Partial<SiteMapFilter>, nodes: readonly SiteMapNode[], more = false): void {
    siteMapFake.answers.set(keyOf({ siteId: "s1", ...filter }), { view: { trail: [], nodes: { items: [...nodes], hasMore: more } }, more });
}

export function fakeSiteMap(filter: SiteMapFilter, enabled = true): unknown {
    const key = `${keyOf(filter)}:${String(enabled)}`;
    if (enabled) {
        siteMapFake.asked.push(filter);
    }
    const found = enabled ? siteMapFake.answers.get(keyOf(filter)) : undefined;
    if (!siteMapFake.stable.has(key)) {
        siteMapFake.stable.set(key, {
            data: found === undefined ? undefined : { pages: [found.view] },
            isPending: found === undefined,
            isError: false,
            error: null,
            hasNextPage: found?.more ?? false,
            isFetchingNextPage: false,
            fetchNextPage: () => {
                siteMapFake.more.push(filter);
            },
        });
    }
    return siteMapFake.stable.get(key);
}
