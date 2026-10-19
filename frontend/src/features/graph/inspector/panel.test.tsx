import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import { aPage } from "../../../testing/pages.js";
import { renderScreen } from "../../../testing/render.js";
import { buildGraphIndex } from "../model/index.js";
import { entity, pageType, typedPage } from "../model/fixture.js";

const updates: unknown[] = [];

function mutation(record: unknown[] | null) {
    return () => ({
        mutate: (request: unknown) => {
            record?.push(request);
        },
        mutateAsync: async () => undefined,
        reset: () => undefined,
        isPending: false,
        error: null,
    });
}

vi.mock("../../../data/hooks/graph.js", () => ({
    useUpdateEntity: mutation(updates),
    useSetAnchors: mutation(null),
    useApproveEdge: mutation(null),
    useRejectEdge: mutation(null),
}));

vi.mock("../../../data/hooks/pages.js", () => ({
    usePage: () => ({ data: { page: aPage("page-peptides", "/peptides/", { status: "published" }) }, isPending: false }),
    usePages: () => ({ data: { pages: [] }, isPending: false, hasNextPage: false }),
    useSetCanonicalPage: mutation(null),
    useMapPageToEntity: mutation(null),
}));

vi.mock("../../agent/index.js", () => ({
    askAgent: () => undefined,
}));

const { Inspector } = await import("./panel.js");

const said = copy.graph.entityCard;

function show(definition = "") {
    const peptides = { ...entity({ id: "peptides", name: "Peptides" }), definition };
    const index = buildGraphIndex([peptides], [], [typedPage("peptides", "hub", "published")], [pageType("hub", "Category")]);
    return renderScreen(
        <Inspector
            siteId="s1"
            index={index}
            selectedId="peptides"
            onSelect={() => undefined}
            onReveal={() => undefined}
            onLens={() => undefined}
            onConnect={() => undefined}
            onDelete={() => undefined}
            onPlanPage={() => undefined}
            onRecompute={() => undefined}
            recomputing={false}
            audit={null}
        />,
    );
}

describe("the inspector's page type and card", () => {
    it("names the canonical page's type by the client's word and its role, and links to it", () => {
        show();

        const line = screen.getByText(copy.graph.inspector.type).closest("[data-entity-page-type]");
        expect(line?.getAttribute("data-entity-page-type")).toBe("hub");
        expect(line?.textContent).toContain(`Category · ${copy.pageTypes.presets.hub}`);
        const link = within(line as HTMLElement).getByRole("link", { name: copy.graph.inspector.typeOpen });
        expect(link.getAttribute("href")).toBe("/settings/types?site=s1&type=type-hub");
    });

    it("refuses a fact with a blank half before anything is sent, then saves the card", () => {
        updates.length = 0;
        show("A family of short chains of amino acids.");

        const card = within(screen.getByRole("form", { name: said.title }));
        fireEvent.click(card.getByRole("button", { name: said.addFact }));
        fireEvent.change(card.getByRole("textbox", { name: `${said.factLabel} 1` }), { target: { value: "Storage" } });
        fireEvent.click(card.getByRole("button", { name: said.save }));
        expect(card.getByText(said.halfBlank)).toBeDefined();
        expect(updates).toHaveLength(0);

        fireEvent.change(card.getByRole("textbox", { name: `${said.factValue} 1` }), { target: { value: "Keep it cold" } });
        fireEvent.click(card.getByRole("button", { name: said.save }));
        expect(updates).toStrictEqual([
            { id: "peptides", definition: null, facts: [{ label: "Storage", value: "Keep it cold" }], notes: null },
        ]);
    });

    it("edits the definition and a note", () => {
        updates.length = 0;
        show();

        const card = within(screen.getByRole("form", { name: said.title }));
        fireEvent.change(card.getByRole("textbox", { name: new RegExp(`^${said.definition}`) }), { target: { value: "Short amino-acid chains." } });
        fireEvent.click(card.getByRole("button", { name: said.addNote }));
        fireEvent.change(card.getByRole("textbox", { name: `${said.noteLabel} 1` }), { target: { value: "Claims" } });
        fireEvent.change(card.getByRole("textbox", { name: `${said.noteText} 1` }), { target: { value: "No medical claims." } });
        fireEvent.click(card.getByRole("button", { name: said.save }));
        expect(updates).toStrictEqual([
            { id: "peptides", definition: "Short amino-acid chains.", facts: null, notes: [{ label: "Claims", text: "No medical claims." }] },
        ]);
    });
});
