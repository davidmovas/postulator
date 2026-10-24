import { fireEvent, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { aPage } from "../../testing/pages.js";
import { aPageType } from "../../testing/pagetypes.js";
import { renderScreen } from "../../testing/render.js";

const said = copy.pages.detail;

const sent: unknown[] = [];

vi.mock("../../data/hooks/pages.js", () => ({
    useSetPageType: () => ({
        mutate: (request: unknown) => {
            sent.push(request);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
}));

vi.mock("../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({
        data: {
            types: [
                aPageType("t-hub", "hub", { name: "Category" }),
                aPageType("t-guide", "entity", { name: "Guide" }),
            ],
            fields: [],
        },
        isPending: false,
    }),
}));

const { TypePanel } = await import("./type.js");

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

describe("a page's type in the drawer", () => {
    it("names the type with its role, and says who set it and why", () => {
        renderScreen(<TypePanel page={aPage("p1", "/peptides/", { pageTypeId: "t-hub", typeSource: "auto", typeReason: "listing" })} siteId="s1" />);

        const label = said.typeOption("Category", copy.pageTypes.presets.hub);
        expect(screen.getAllByText(label)).toHaveLength(2);
        expect(screen.getByRole("combobox", { name: said.type }).textContent).toContain(label);
        expect(screen.getByText(copy.pageTypes.sources.auto)).toBeDefined();
        expect(
            screen.getByText(said.typeWhy(copy.pageTypes.sources.auto, copy.pageTypes.reasons.listing)),
        ).toBeDefined();
        expect(screen.getByRole("button", { name: said.typeSave })).toHaveProperty("disabled", true);
    });

    it("says a page chosen by a person keeps its type", () => {
        renderScreen(<TypePanel page={aPage("p1", "/guide/", { pageTypeId: "t-guide", typeSource: "manual", typeReason: "user" })} siteId="s1" />);

        expect(screen.getByText(said.typeWhy(copy.pageTypes.sources.manual, copy.pageTypes.reasons.user))).toBeDefined();
        expect(screen.getByText(said.typeHint)).toBeDefined();
    });

    it("waits for the next sync to type an untyped page", () => {
        renderScreen(<TypePanel page={aPage("p1", "/new/")} siteId="s1" />);

        expect(screen.getByText(said.typeNone)).toBeDefined();
        expect(screen.getByText(copy.pageTypes.sources[""])).toBeDefined();
    });

    it("says so when the page names a type the site no longer has", () => {
        renderScreen(<TypePanel page={aPage("p1", "/old/", { pageTypeId: "t-gone", typeSource: "manual", typeReason: "ai" })} siteId="s1" />);

        expect(screen.getByText(said.typeUnknown)).toBeDefined();
    });

    it("sets the type a person picks through SetType", () => {
        sent.length = 0;
        renderScreen(<TypePanel page={aPage("p1", "/guide/", { pageTypeId: "t-guide", typeSource: "auto", typeReason: "default" })} siteId="s1" />);

        fireEvent.pointerDown(screen.getByRole("combobox", { name: said.type }), { button: 0, ctrlKey: false, pointerType: "mouse" });
        fireEvent.click(screen.getByRole("option", { name: said.typeOption("Category", copy.pageTypes.presets.hub) }));
        fireEvent.click(screen.getByRole("button", { name: said.typeSave }));

        expect(sent).toStrictEqual([{ pageId: "p1", pageTypeId: "t-hub" }]);
    });
});
