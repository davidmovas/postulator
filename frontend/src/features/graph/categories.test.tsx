import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Entity } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { entity } from "./model/fixture.js";
import { expandAll, visibleRows } from "./model/fold.js";
import { buildGraphIndex } from "./model/index.js";

vi.mock("./inspector/page.js", () => ({ CanonicalPage: () => null }));
vi.mock("./inspector/fields.js", () => ({ EntityFields: () => null }));
vi.mock("./inspector/anchors.js", () => ({ AnchorsEditor: () => null }));
vi.mock("./inspector/relations.js", () => ({ Relations: () => null }));

const { Inspector } = await import("./inspector/panel.js");
const { NodeCard } = await import("./node-card.js");
const { OutlineView } = await import("./outline/view.js");

const filed: Entity["categories"] = [
    { id: "peptides", name: "Peptides", termId: 12 },
    { id: "healing", name: "Healing" },
];

function graphOf(categories: Entity["categories"]) {
    return buildGraphIndex(
        [entity({ id: "bpc", name: "BPC-157", kind: "product" }), entity({ id: "tb", name: "TB-500" })].map((held) =>
            held.id === "bpc" ? { ...held, categories } : held,
        ),
        [],
    );
}

function statesIn(container: HTMLElement): (string | undefined)[] {
    const trail = within(container).getByRole("list", { name: copy.categories.trail });
    return [...trail.querySelectorAll("li")].map((item) => item.dataset["categoryState"]);
}

function inspect(categories: Entity["categories"]): void {
    renderScreen(
        <Inspector
            siteId="s1"
            index={graphOf(categories)}
            selectedId="bpc"
            onSelect={() => {}}
            onReveal={() => {}}
            onLens={() => {}}
            onConnect={() => {}}
            onDelete={() => {}}
            onPlanPage={() => {}}
            onRecompute={() => {}}
            recomputing={false}
            audit={null}
        />,
    );
}

describe("the inspector header", () => {
    it("names the categories the entity's page is filed under, and which WordPress holds", () => {
        inspect(filed);
        const strip = document.querySelector<HTMLElement>("[data-entity-categories]");
        if (strip === null) {
            throw new Error("the inspector shows no categories");
        }
        expect(within(strip).getByText(copy.graph.inspector.filedUnder)).toBeDefined();
        expect(statesIn(strip)).toStrictEqual(["onSite", "onPublish"]);
        expect(within(strip).getByText("Peptides").closest("li")?.getAttribute("title")).toBe(copy.categories.onSite(12));
    });

    it.each([[[]], [null]])("shows nothing for an entity whose page is filed under nothing (%j)", (categories) => {
        inspect(categories);
        expect(document.querySelector("[data-entity-categories]")).toBeNull();
        expect(screen.getByText("BPC-157")).toBeDefined();
    });
});

describe("the node card", () => {
    it("shows the trail of the hovered entity's page", () => {
        renderScreen(<NodeCard index={graphOf(filed)} hovered={{ id: "bpc", at: { x: 10, y: 10 } }} hostWidth={800} />);
        expect(statesIn(screen.getByRole("tooltip"))).toStrictEqual(["onSite", "onPublish"]);
    });

    it("leaves the trail out for an entity with no filed page", () => {
        renderScreen(<NodeCard index={graphOf(filed)} hovered={{ id: "tb", at: { x: 10, y: 10 } }} hostWidth={800} />);
        expect(within(screen.getByRole("tooltip")).queryByRole("list", { name: copy.categories.trail })).toBeNull();
    });
});

describe("the outline", () => {
    it("gives each entity a Page categories column with a one-line trail", () => {
        const index = graphOf(filed);
        renderScreen(
            <OutlineView
                siteId="s1"
                index={index}
                rows={visibleRows(index, expandAll(index), "score")}
                selectedId={null}
                matched={null}
                onSelect={() => {}}
                onPick={() => {}}
                onToggle={() => {}}
                onLiftMore={() => {}}
            />,
        );
        expect(screen.getByText(copy.graph.outline.columns.categories)).toBeDefined();
        const row = screen.getByText("BPC-157").closest<HTMLElement>('[role="row"]');
        const other = screen.getByText("TB-500").closest<HTMLElement>('[role="row"]');
        if (row === null || other === null) {
            throw new Error("the outline has no row for the entities");
        }
        expect(statesIn(row)).toStrictEqual(["onSite", "onPublish"]);
        const trail = within(row).getByRole("list", { name: copy.categories.trail });
        expect(trail.className).toContain("flex-nowrap");
        expect(trail.getAttribute("title")).toContain("Peptides › Healing");
        expect(within(other).queryByRole("list", { name: copy.categories.trail })).toBeNull();
    });
});
