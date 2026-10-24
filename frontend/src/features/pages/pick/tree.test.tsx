import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { SiteMapNode } from "../../../data/types.js";
import { answerSiteMap, resetSiteMap, siteMapFake } from "../../../testing/site-map.js";
import { aNode, aRef } from "../../map/model/fixture.js";
import type { Keep, NamedPage } from "./model.js";
import { everyNode } from "./model.js";

vi.mock("../../../data/hooks/pages.js", async () => {
    const fake = await import("../../../testing/site-map.js");
    return { useSiteMap: fake.fakeSiteMap };
});

const { PageTree } = await import("./tree.js");

const said = copy.pages.pick;

const mapped = (node: SiteMapNode): boolean => (node.entityId ?? "") !== "";

const shop = aNode("shop", "/shop/", { entityId: "e1", children: 2, descendants: 3 });
const about = aNode("about", "/about/", { entityId: "e2" });
const mugs = aNode("mugs", "/shop/mugs/", { entityId: "e3", page: aRef("mugs", "/shop/mugs/", { onSite: false }) });
const bags = aNode("bags", "/shop/bags/");
const travel = aNode("travel", "/shop/mugs/travel/", { entityId: "e4" });

const named: NamedPage[][] = [];

function Harness({ keep = everyNode }: { keep?: Keep }) {
    const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
    return (
        <div>
            <span data-selected={[...selected].sort().join(",")} />
            <PageTree
                siteId="s1"
                selected={selected}
                onChange={(next, pages) => {
                    named.push([...pages]);
                    setSelected(next);
                }}
                pickable={mapped}
                keep={keep}
                label="Pages"
                picked={`${selected.size} picked`}
                refusalOf={(node) => (mapped(node) ? null : "Nothing is mapped to it")}
            />
        </div>
    );
}

function rowOf(pageId: string): HTMLElement {
    const row = document.querySelector<HTMLElement>(`[data-page-pick="${pageId}"]`);
    if (row === null) {
        throw new Error(`no row for ${pageId}`);
    }
    return row;
}

function selectedNow(): string {
    return document.querySelector("[data-selected]")?.getAttribute("data-selected") ?? "";
}

beforeEach(() => {
    resetSiteMap();
    named.length = 0;
    answerSiteMap({}, [shop, about]);
    answerSiteMap({ parentId: "shop" }, [mugs, bags]);
    answerSiteMap({ search: "travel" }, [travel]);
});

describe("the page picker over the site map", () => {
    it("reads the top of the map, and a branch only once it is opened", () => {
        render(<Harness />);

        expect(screen.getByRole("checkbox", { name: "shop/" })).toBeTruthy();
        expect(screen.queryByRole("checkbox", { name: "mugs/" })).toBeNull();
        expect(siteMapFake.asked.some((filter) => filter.parentId === "shop")).toBe(false);

        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.expand }));

        expect(siteMapFake.asked.some((filter) => filter.parentId === "shop")).toBe(true);
        expect(screen.getByRole("checkbox", { name: "mugs/" })).toBeTruthy();
    });

    it("keeps a pick when its branch closes and when the search changes", async () => {
        render(<Harness />);
        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.expand }));
        fireEvent.click(screen.getByRole("checkbox", { name: "mugs/" }));

        expect(selectedNow()).toBe("mugs");
        expect(named.at(-1)).toStrictEqual([{ id: "mugs", path: "/shop/mugs/" }]);

        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.collapse }));
        expect(screen.queryByRole("checkbox", { name: "mugs/" })).toBeNull();
        expect(screen.getByRole("checkbox", { name: "shop/" }).getAttribute("aria-checked")).toBe("mixed");

        fireEvent.change(screen.getByRole("searchbox", { name: said.search }), { target: { value: "travel" } });
        await waitFor(() => {
            expect(screen.getByRole("checkbox", { name: "/shop/mugs/travel/" })).toBeTruthy();
        });
        fireEvent.click(screen.getByRole("checkbox", { name: "/shop/mugs/travel/" }));
        expect(selectedNow()).toBe("mugs,travel");

        fireEvent.change(screen.getByRole("searchbox", { name: said.search }), { target: { value: "" } });
        await waitFor(() => {
            expect(screen.getByRole("checkbox", { name: "shop/" })).toBeTruthy();
        });
        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.expand }));
        expect((screen.getByRole("checkbox", { name: "mugs/" }) as HTMLInputElement).checked).toBe(true);
        expect(selectedNow()).toBe("mugs,travel");
    });

    it("refuses a page nothing is mapped to, in words", () => {
        render(<Harness />);
        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.expand }));

        expect((screen.getByRole("checkbox", { name: "bags/" }) as HTMLInputElement).disabled).toBe(true);
        expect(rowOf("bags").getAttribute("title")).toBe("Nothing is mapped to it");
    });

    it("asks for the rest of a branch through its cursor", () => {
        answerSiteMap({}, [shop, about], true);
        render(<Harness />);

        fireEvent.click(screen.getByRole("button", { name: said.more }));

        expect(siteMapFake.more).toStrictEqual([{ siteId: "s1" }]);
    });

    it("narrows the loaded rows by the filter and keeps a branch that can still be opened", () => {
        render(<Harness keep={(node) => !node.page.onSite} />);

        expect(screen.getByRole("checkbox", { name: "shop/" })).toBeTruthy();
        expect(screen.queryByRole("checkbox", { name: "about/" })).toBeNull();
        fireEvent.click(within(rowOf("shop")).getByRole("button", { name: said.expand }));
        expect(screen.getByRole("checkbox", { name: "mugs/" })).toBeTruthy();
        expect(screen.queryByRole("checkbox", { name: "bags/" })).toBeNull();
    });
});
