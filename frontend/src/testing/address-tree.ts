import type { AddressFilter, AddressNode, AddressView, SiteMapNode } from "../data/types.js";

interface Answer {
    view: AddressView;
    more: boolean;
}

export interface AddressTreeFake {
    answers: Map<string, Answer>;
    asked: AddressFilter[];
    stable: Map<string, unknown>;
}

export const addressTreeFake: AddressTreeFake = { answers: new Map(), asked: [], stable: new Map() };

function keyOf(filter: AddressFilter): string {
    return JSON.stringify({ folder: filter.folder ?? "", within: filter.within ?? "" });
}

export function resetAddressTree(): void {
    addressTreeFake.answers.clear();
    addressTreeFake.stable.clear();
    addressTreeFake.asked.length = 0;
}

export function aFolder(path: string, children: number, pages: number, page?: SiteMapNode): AddressNode {
    return page === undefined ? { path, children, pages } : { path, children, pages, page };
}

export function answerAddressTree(filter: Partial<AddressFilter>, nodes: readonly AddressNode[], more = false): void {
    const held = { siteId: "s1", ...filter };
    addressTreeFake.answers.set(keyOf(held), { view: { folder: held.folder ?? "", nodes: { items: [...nodes], hasMore: more } }, more });
}

export function fakeAddressTree(filter: AddressFilter, enabled = true): unknown {
    const key = `${keyOf(filter)}:${String(enabled)}`;
    if (enabled) {
        addressTreeFake.asked.push(filter);
    }
    const found = enabled ? addressTreeFake.answers.get(keyOf(filter)) : undefined;
    if (!addressTreeFake.stable.has(key)) {
        addressTreeFake.stable.set(key, {
            data: found === undefined ? undefined : { pages: [found.view] },
            isPending: found === undefined,
            isError: false,
            error: null,
            hasNextPage: found?.more ?? false,
            isFetchingNextPage: false,
            fetchNextPage: () => undefined,
        });
    }
    return addressTreeFake.stable.get(key);
}
