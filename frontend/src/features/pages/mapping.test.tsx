import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { aPage } from "../../testing/pages.js";
import { renderScreen } from "../../testing/render.js";
import type { EntityIndex } from "./entities.js";

const said = copy.pages.detail;

function idle() {
    return { mutate: () => undefined, reset: () => undefined, isPending: false, error: null };
}

vi.mock("../../data/hooks/graph.js", () => ({
    useEntity: (id: string | null) =>
        id === null
            ? { data: undefined, isPending: false }
            : { data: { entity: { id, name: "Ceramic Mugs", score: 0.4, canonicalPageId: "p1" } }, isPending: false },
}));

vi.mock("../../data/hooks/pages.js", () => ({
    useMapPageToEntity: idle,
    useUnmapPage: idle,
    useSetCanonicalPage: idle,
}));

const { PageMapping } = await import("./mapping.js");

const index: EntityIndex = { entities: [], byId: new Map(), labels: new Map(), complete: true, loading: false };

describe("a page's graph entity", () => {
    it("says a page without one needs an entity its type does not give, and leads to the proposals", () => {
        renderScreen(<PageMapping page={aPage("p1", "/mugs/", { entityId: null })} siteId="s1" index={index} search="" />);

        expect(screen.getByRole("heading", { name: said.entity })).toBeDefined();
        expect(screen.getByRole("link", { name: said.entityWhy }).getAttribute("href")).toBe("/s/s1/findings?code=page_without_entity");
    });

    it("keeps the same sentence, without a link, on a page the graph already holds", () => {
        renderScreen(<PageMapping page={aPage("p1", "/mugs/", { entityId: "e1" })} siteId="s1" index={index} search="" />);

        expect(screen.getByText(said.entityWhy)).toBeDefined();
        expect(screen.queryByRole("link", { name: said.entityWhy })).toBeNull();
        expect(screen.getByText("Ceramic Mugs")).toBeDefined();
    });
});
