import { describe, expect, it } from "vitest";

import type { AddressNode } from "../../../data/types.js";
import { aNode } from "../../map/model/fixture.js";
import type { AddressBranch, AddressBranches } from "./model.js";
import { addressChildren, addressRow, asMapNode, flattenAddress, keyOf, openFolders } from "./model.js";

function folder(path: string, children: number, pages: number, pageId?: string, type = ""): AddressNode {
    if (pageId === undefined) {
        return { path, children, pages };
    }
    const page = aNode(pageId, path);
    return { path, children, pages, page: { ...page, page: { ...page.page, type } } };
}

const home = folder("/", 2, 4, "home");
const products = folder("/products/", 2, 2);
const guides = folder("/guides/", 0, 1, "guides", "Guide");
const p1 = folder("/products/p1/", 0, 1, "p1", "Product");
const p2 = folder("/products/p2/", 0, 1, "p2", "Product");

function branches(entries: readonly (readonly [string, Partial<AddressBranch>])[]): AddressBranches {
    return new Map(entries.map(([key, value]) => [key, { nodes: [], hasMore: false, pending: false, ...value }]));
}

const loaded = branches([
    ["", { nodes: [home] }],
    ["/", { nodes: [guides, products] }],
    ["/products/", { nodes: [p1, p2] }],
]);

const typeIdOf = (name: string): string | null => (name === "" ? null : `type-${name}`);

describe("the pages tree by address", () => {
    it("keys a folder with a page by the page and one without by its path", () => {
        expect([home, products, p1].map(keyOf)).toStrictEqual(["home", "/products/", "p1"]);
    });

    it("asks for an expanded folder's children by its path", () => {
        expect(openFolders(new Set(["/", "/products/"]), loaded)).toStrictEqual(["", "/", "/products/"]);
        expect(openFolders(new Set(["/products/"]), loaded)).toStrictEqual([""]);
    });

    it("lays the folders out by depth, each with the keys above it", () => {
        const rows = flattenAddress(new Set(["/", "/products/"]), loaded).flatMap((row) =>
            row.kind === "node" ? [[row.node.path, row.depth, row.ancestors.join(",")]] : [],
        );
        expect(rows).toStrictEqual([
            ["/", 0, ""],
            ["/guides/", 1, "home"],
            ["/products/", 1, "home"],
            ["/products/p1/", 2, "/products/,home"],
            ["/products/p2/", 2, "/products/,home"],
        ]);
    });

    it("shows a branch still loading and one with more to read", () => {
        const kinds = flattenAddress(new Set(["/"]), branches([["", { nodes: [home], hasMore: true }]])).map((row) => row.kind);
        expect(kinds).toStrictEqual(["node", "loading", "more"]);
    });

    it("counts a folder's pages and children on the page row it draws", () => {
        expect(asMapNode(products)).toBeNull();
        const drawn = asMapNode(home);
        expect([drawn?.children, drawn?.descendants]).toStrictEqual([2, 3]);
    });

    it("hands the selection a folder's children with their sizes and types", () => {
        const folders = new Map([["home", "/"], ["/products/", "/products/"]]);
        const chains = new Map<string, readonly string[]>([["home", []], ["/products/", ["home"]]]);
        const childrenOf = addressChildren(loaded, (key) => folders.get(key), (key) => chains.get(key), typeIdOf, undefined);
        expect(childrenOf("/products/")).toStrictEqual([
            { id: "p1", row: { ancestors: ["/products/", "home"], typeId: "type-Product" }, size: 1 },
            { id: "p2", row: { ancestors: ["/products/", "home"], typeId: "type-Product" }, size: 1 },
        ]);
        expect(childrenOf("p1")).toBeNull();
        expect(addressRow(products, ["home"], typeIdOf)).toStrictEqual({ ancestors: ["home"], typeId: null });
    });
});
