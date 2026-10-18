import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { PageModel } from "../../../data/types.js";
import { renderScreen } from "../../../testing/render.js";
import { frontModel, hubModel, plannedModel, unrenderedModel } from "./fixture.js";

const held: { model: PageModel | undefined } = { model: hubModel };

vi.mock("../../../data/hooks/pages.js", () => ({
    usePageModel: () => ({ data: held.model, isPending: held.model === undefined, error: null }),
}));

const { PageModelTab } = await import("./tab.js");

const said = copy.pages.model;

function tab(model: PageModel) {
    held.model = model;
    return renderScreen(<PageModelTab pageId={model.pageId} siteId="s1" search="" />);
}

function pane(title: string): HTMLElement {
    const heading = screen.getByRole("heading", { name: title });
    const section = heading.closest("section");
    if (section === null) {
        throw new Error(`no pane titled ${title}`);
    }
    return section;
}

describe("the page model tab", () => {
    it("shows where the page lives on the site", () => {
        tab(hubModel);

        const identity = within(pane(said.identity.title));
        expect(identity.getByRole("button", { name: /https:\/\/shop\.test\/tb-500\// })).toBeDefined();
        expect(identity.getByText(said.identity.typeWithId("Page", 412))).toBeDefined();
        expect(identity.getByText("Published")).toBeDefined();
        expect(identity.getByText("Healing Peptides")).toBeDefined();
    });

    it("links the parent a redirect resolved and says how it was found", () => {
        tab(hubModel);

        const parent = within(pane(said.hierarchy.title));
        const link = parent.getByRole("link", { name: /Peptides/ });
        expect(link.getAttribute("href")).toBe("/s/s1/pages/p-peptides");
        expect(parent.getByText("from the JSON-LD breadcrumb, through a redirect")).toBeDefined();
    });

    it("lays every trail out crumb by crumb with what became of each", () => {
        tab(hubModel);

        const trails = pane(said.trails.title);
        const jsonld = within(within(trails).getByRole("region", { name: "JSON-LD breadcrumb list" }));
        expect(jsonld.getByText(said.trails.redirects("/peptides/"))).toBeDefined();
        expect(jsonld.getByText(said.trails.self)).toBeDefined();
        expect(jsonld.getByRole("link", { name: /Peptides/ }).getAttribute("href")).toBe("/s/s1/pages/p-peptides");

        const nav = within(within(trails).getByRole("region", { name: "Breadcrumb navigation" }));
        expect(nav.getByText(said.trails.noLink)).toBeDefined();
        expect(nav.getByText(said.trails.status(404))).toBeDefined();
        expect(nav.getAllByText(said.trails.unresolved)).toHaveLength(2);
    });

    it("names the content root, the words and every region in words", () => {
        tab(hubModel);

        const rendered = within(pane(said.render.title));
        expect(rendered.getByText("Content found")).toBeDefined();
        expect(rendered.getByText(said.render.wordCounts(9, 240))).toBeDefined();
        expect(rendered.getByText("Dynamic content")).toBeDefined();
        expect(rendered.getAllByText(said.render.outside)).toHaveLength(3);
        expect(rendered.getByText("Absolute")).toBeDefined();
    });

    it("types every link, links the pages it points to and marks the one a run wrote", () => {
        tab(hubModel);

        const links = within(pane(said.links.title));
        expect(links.getByText(said.links.count(3, 1))).toBeDefined();
        expect(links.getByText("Contextual")).toBeDefined();
        expect(links.getByText("Content · in the body")).toBeDefined();
        expect(links.getByRole("link", { name: /\/bpc-157\// }).getAttribute("href")).toBe("/s/s1/pages/p-bpc");
        expect(links.getAllByLabelText(said.links.ours)).toHaveLength(1);
    });

    it("lists the blocks read only and marks the dynamic one and ours", () => {
        tab(hubModel);

        const blocks = within(pane(said.blocks.title));
        expect(blocks.getByText("Classic editor")).toBeDefined();
        expect(blocks.getByText("Shortcode")).toBeDefined();
        expect(blocks.getByText(said.blocks.dynamic)).toBeDefined();
        expect(blocks.getAllByText(said.blocks.ours)).toHaveLength(1);
        expect(blocks.queryByRole("button")).toBeNull();
        expect(blocks.queryByRole("textbox")).toBeNull();
    });

    it("words every open finding from its subject and offers no action", () => {
        tab(hubModel);

        const findings = within(pane(said.findings.title));
        expect(findings.getByText("Crumb without a page")).toBeDefined();
        expect(findings.getByText(said.findings.unresolved("Healing Peptides", ""))).toBeDefined();
        expect(findings.getByText(said.findings.ambiguous("Guides", ["/lab-guides/", "/reconstitution/guides/"]))).toBeDefined();
        expect(findings.queryByRole("button")).toBeNull();
    });

    it("prints no stored vocabulary value anywhere", () => {
        const { container } = tab(hubModel);

        for (const raw of ["jsonld_breadcrumb", "content_dynamic", "parent_unresolved", "parent_ambiguous", "product_cat", "{"]) {
            expect(container.textContent).not.toContain(raw);
        }
    });

    it("says a planned page is not on the site yet and keeps its workbook parent", () => {
        tab(plannedModel);

        expect(screen.getByText(said.notOnSite)).toBeDefined();
        expect(screen.getByText("from the workbook")).toBeDefined();
        expect(screen.getByText(said.findings.none)).toBeDefined();
        expect(screen.queryByRole("heading", { name: said.render.title })).toBeNull();
    });

    it("says a page on the site has no render yet and leaves the trails out", () => {
        tab(unrenderedModel);

        expect(screen.getByText(said.render.none)).toBeDefined();
        expect(screen.queryByRole("heading", { name: said.trails.title })).toBeNull();
        expect(screen.getByText(said.links.none)).toBeDefined();
        expect(screen.getByText(said.blocks.empty)).toBeDefined();
        expect(screen.getByText(said.findings.none)).toBeDefined();
    });

    it("reads the front page as the root of the site and words its read-only reason", () => {
        tab(frontModel);

        expect(screen.getByText(said.hierarchy.front)).toBeDefined();
        expect(screen.getByText("Made with elementor, read only")).toBeDefined();
    });
});
