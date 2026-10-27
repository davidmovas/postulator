import { fireEvent, screen, within } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { PageType, WriterDirections, WriterPack } from "../../../data/types.js";
import { aPage } from "../../../testing/pages.js";
import { aPageType, aScope } from "../../../testing/pagetypes.js";
import { renderScreen } from "../../../testing/render.js";

const said = copy.pages.writer;
const fields = copy.pageTypes.fields;

interface Held {
    pack: WriterPack | undefined;
    error: unknown;
    asked: string[];
    directions: (WriterDirections | null)[];
    types: PageType[];
}

const held: Held = { pack: undefined, error: null, asked: [], directions: [], types: [] };

vi.mock("../../../data/hooks/pages.js", () => ({
    useWriterPack: (_pageId: string | null, field: string, directions: WriterDirections | null = null) => {
        held.asked.push(field);
        held.directions.push(directions);
        return { data: held.pack, isPending: held.pack === undefined && held.error === null, error: held.error };
    },
}));

vi.mock("../../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types: held.types, fields: [] }, isPending: false }),
}));

const { WriterTab } = await import("./pane.js");

function hubType(): PageType {
    return aPageType("t-hub", "hub", {
        name: "Category",
        scopes: [
            aScope("body", { enabled: true }),
            aScope("seo_title", { enabled: true }),
            aScope("seo_description", { enabled: false }),
            aScope("excerpt", { enabled: true }),
            aScope("image_alt", { enabled: true }),
        ],
    });
}

function fitted(overrides: Partial<WriterPack> = {}): WriterPack {
    return {
        pageId: "p1",
        path: "/peptides/",
        field: "body",
        budget: 900,
        built: 1240,
        tokens: 860,
        sections: [
            {
                name: "page",
                heading: "The page",
                priority: 1,
                lines: ["Path: /peptides/", "Title: Peptides"],
                items: [],
                tokens: 40,
            },
            {
                name: "siblings",
                heading: "Siblings, the pages beside it",
                priority: 5,
                lines: [],
                items: ["/nootropics/ Nootropics"],
                tokens: 60,
            },
            {
                name: "trail",
                heading: "The trail above it",
                priority: 3,
                lines: [],
                items: ["/ Home", "/shop/ Shop"],
                tokens: 30,
            },
            {
                name: "rules",
                heading: "The field's rules",
                priority: 1,
                lines: ["Field: Body", "Links: at most 20 links"],
                items: [],
                tokens: 25,
            },
        ],
        dropped: [
            { section: "children", heading: "Children, the pages below it", items: 0, tokens: 300, whole: true },
            { section: "products", heading: "Products it lists", items: 4, tokens: 80, whole: false },
        ],
        ...overrides,
    };
}

function tab(type: PageType | null = hubType()) {
    held.types = type === null ? [] : [type];
    return renderScreen(<WriterTab page={aPage("p1", "/peptides/", { pageTypeId: type?.id ?? null })} siteId="s1" />);
}

function sectionNamed(name: string): HTMLElement {
    const found = document.querySelector<HTMLElement>(`[data-pack-section="${name}"]`);
    if (found === null) {
        throw new Error(`no section ${name}`);
    }
    return found;
}

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

beforeEach(() => {
    held.pack = fitted();
    held.error = null;
    held.asked = [];
    held.directions = [];
});

describe("what the writer will see with a run's directions", () => {
    const directed = fitted({
        sections: [
            {
                name: "brief",
                heading: "The run's brief",
                priority: 1,
                lines: ["Goal: Sell the starter kit", "Link to /kits/ with the anchor “peptide kits”"],
                items: [],
                tokens: 30,
            },
            {
                name: "reference",
                heading: "Reference text the person gave for this run",
                priority: 2,
                lines: [],
                items: ["Kits ship within two days."],
                tokens: 12,
            },
            {
                name: "style",
                heading: "A sample of the style to write in",
                priority: 2,
                lines: ["Short sentences. No jargon."],
                items: [],
                tokens: 8,
            },
        ],
    });

    it("asks for the pack with the draft brief and directive and says so", () => {
        held.pack = directed;
        held.types = [hubType()];
        const directions = { brief: { goal: "Sell the starter kit" }, directive: { pageId: "p1", notes: "Lead with it" } };
        renderScreen(
            <WriterTab
                page={aPage("p1", "/peptides/", { pageTypeId: "t-hub" })}
                siteId="s1"
                directions={directions}
                context="draft"
            />,
        );

        expect(held.directions.at(-1)).toStrictEqual(directions);
        expect(screen.getByText(said.directed.draft)).toBeDefined();
    });

    it("titles the brief, the reference and the style sample in plain words", () => {
        held.pack = directed;
        held.types = [hubType()];
        renderScreen(
            <WriterTab
                page={aPage("p1", "/peptides/", { pageTypeId: "t-hub" })}
                siteId="s1"
                directions={{ runId: "r1" }}
                context="run"
            />,
        );

        expect(within(sectionNamed("brief")).getByRole("heading", { name: said.sectionTitles["brief"] })).toBeDefined();
        expect(within(sectionNamed("brief")).getByText("Goal: Sell the starter kit")).toBeDefined();
        expect(within(sectionNamed("reference")).getByRole("heading", { name: said.sectionTitles["reference"] })).toBeDefined();
        expect(within(sectionNamed("style")).getByText("Short sentences. No jargon.")).toBeDefined();
        expect(screen.getByText(said.directed.run)).toBeDefined();
    });
});

describe("what the writer will see", () => {
    it("asks for the page's link field by default and names it", () => {
        tab();

        expect(held.asked.every((field) => field === "")).toBe(true);
        expect(screen.getByRole("combobox", { name: said.field }).textContent).toContain(fields.body);
    });

    it("offers only the enabled fields the writer writes", () => {
        tab();

        fireEvent.pointerDown(screen.getByRole("combobox", { name: said.field }), { button: 0, ctrlKey: false, pointerType: "mouse" });
        const offered = screen.getAllByRole("option").map((option) => option.textContent);
        expect(offered).toEqual([fields.body, fields.excerpt, fields.seo_title]);
    });

    it("asks for the field a person picks", () => {
        tab();

        fireEvent.pointerDown(screen.getByRole("combobox", { name: said.field }), { button: 0, ctrlKey: false, pointerType: "mouse" });
        fireEvent.click(screen.getByRole("option", { name: fields.seo_title }));

        expect(held.asked.at(-1)).toBe("seo_title");
    });

    it("defaults a product to its description", () => {
        tab(aPageType("t-product", "product", { scopes: [aScope("product_description", { enabled: true })] }));

        expect(screen.getByRole("combobox", { name: said.field }).textContent).toContain(fields.product_description);
    });

    it("lays the sections out by priority, each with its heading, priority, tokens, lines and items", () => {
        tab();

        const order = [...document.querySelectorAll("[data-pack-section]")].map((node) => node.getAttribute("data-pack-section"));
        expect(order).toEqual(["page", "rules", "trail", "siblings"]);

        const trail = within(sectionNamed("trail"));
        expect(trail.getByRole("heading", { name: "The trail above it" })).toBeDefined();
        expect(trail.getByText(said.priority(3))).toBeDefined();
        expect(trail.getByText(said.tokens("30"))).toBeDefined();
        expect(trail.getByText("/shop/ Shop")).toBeDefined();

        const rules = within(sectionNamed("rules"));
        expect(rules.getByText("Links: at most 20 links")).toBeDefined();
    });

    it("says how much of the pack goes to the model against the budget", () => {
        tab();

        expect(screen.getByRole("meter", { name: said.kept("860", "1,240") })).toBeDefined();
        expect(screen.getByText(said.withinBudget("900"))).toBeDefined();
    });

    it("says the whole window is open when no budget applies", () => {
        held.pack = fitted({ budget: 0, tokens: 1240, dropped: [] });
        tab();

        expect(screen.queryByRole("meter")).toBeNull();
        expect(screen.getByText(said.kept("1,240", "1,240"))).toBeDefined();
        expect(screen.getByText(said.wholeWindow)).toBeDefined();
        expect(screen.getByText(said.dropped.none)).toBeDefined();
    });

    it("lists what the fit left out, whole sections and items from the end", () => {
        tab();

        const children = within(document.querySelector<HTMLElement>('[data-pack-dropped="children"]') ?? document.body);
        expect(children.getByText("Children, the pages below it")).toBeDefined();
        expect(children.getByText(said.dropped.whole)).toBeDefined();
        expect(children.getByText(said.tokens("300"))).toBeDefined();

        const products = within(document.querySelector<HTMLElement>('[data-pack-dropped="products"]') ?? document.body);
        expect(products.getByText(said.dropped.items(4))).toBeDefined();
    });

    it("explains a budget too small to carry the page and links to the settings", () => {
        held.pack = undefined;
        held.error = Object.assign(new Error("the input budget is smaller than what every call must carry"), {
            cause: {
                code: "INVALID",
                message: "the input budget is smaller than what every call must carry",
                details: { reason: "budget_too_small", budget: 50, needed: 210 },
            },
        });
        tab();

        expect(screen.getByText(said.tooSmall("50", "210"))).toBeDefined();
        expect(screen.getByRole("link", { name: said.openSettings }).getAttribute("href")).toBe("/settings/models");
    });

    it("shows any other refusal as it came", () => {
        held.pack = undefined;
        held.error = Object.assign(new Error("the writer never writes this field"), {
            cause: { code: "INVALID", message: "the writer never writes this field" },
        });
        tab();

        expect(screen.getByText("the writer never writes this field")).toBeDefined();
    });

    it("offers the body alone on a page with no type", () => {
        tab(null);

        const select = screen.getByRole("combobox", { name: said.field });
        expect(select.textContent).toContain(fields.body);
        expect(select.hasAttribute("disabled") || select.getAttribute("data-disabled") !== null).toBe(true);
    });
});
