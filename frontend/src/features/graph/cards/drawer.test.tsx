import { fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { CardProposal } from "../../../data/types.js";
import { renderScreen } from "../../../testing/render.js";

const said = copy.graph.cards;

interface Held {
    proposals: CardProposal[];
    accepted: unknown[];
    dismissed: unknown[];
    acceptError: unknown;
}

const held: Held = { proposals: [], accepted: [], dismissed: [], acceptError: null };

function refused(code: string, message: string, details?: Record<string, unknown>): Error {
    return Object.assign(new Error(message), { cause: { code, message, details } });
}

vi.mock("../../../data/hooks/graph.js", () => ({
    useCardProposals: () => ({
        data: { pages: [{ items: held.proposals, hasMore: false }], pageParams: [undefined] },
        isPending: false,
        isError: false,
        error: null,
        hasNextPage: false,
        isFetchingNextPage: false,
        fetchNextPage: async () => undefined,
    }),
    useAcceptCard: () => ({
        mutate: (request: unknown) => {
            held.accepted.push(request);
        },
        reset: () => undefined,
        isPending: false,
        error: held.acceptError,
    }),
    useDismissCard: () => ({
        mutate: (request: unknown) => {
            held.dismissed.push(request);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
}));

const { CardProposalsDrawer } = await import("./drawer.js");

function peptides(overrides: Partial<CardProposal> = {}): CardProposal {
    return {
        findingId: "f-peptides",
        entityId: "e-peptides",
        name: "Peptides",
        current: { definition: "Short chains of amino acids.", facts: [{ label: "Storage", value: "Keep cold" }] },
        proposed: {
            definition: "Short chains of amino acids, sold for laboratory research.",
            facts: [
                { label: "Storage", value: "Keep at 2 to 8 °C" },
                { label: "Purity", value: "Above 98%" },
            ],
        },
        reason: "The category page and its products say so.",
        changed: false,
        createdAt: "2026-10-19T09:00:00Z",
        ...overrides,
    };
}

const revealed: string[] = [];

function drawer(focus: string | null = null) {
    return renderScreen(
        <CardProposalsDrawer
            siteId="s1"
            open={true}
            focus={focus}
            onOpenChange={() => undefined}
            onReveal={(id) => {
                revealed.push(id);
            }}
        />,
    );
}

function proposalOf(entityId: string): HTMLElement {
    const found = document.querySelector<HTMLElement>(`[data-card-proposal="${entityId}"]`);
    if (found === null) {
        throw new Error(`no proposal for ${entityId}`);
    }
    return found;
}

beforeEach(() => {
    held.proposals = [peptides()];
    held.accepted = [];
    held.dismissed = [];
    held.acceptError = null;
    revealed.length = 0;
    Element.prototype.scrollIntoView = () => undefined;
});

describe("the card proposals on the Graph screen", () => {
    it("sets the card now against the proposed one, marking what changes", () => {
        drawer();

        const proposal = within(proposalOf("e-peptides"));
        const now = within(proposal.getByRole("region", { name: said.current }));
        const next = within(proposal.getByRole("region", { name: said.proposed }));

        expect(now.getByText("Short chains of amino acids.")).toBeDefined();
        expect(next.getByText("Short chains of amino acids, sold for laboratory research.")).toBeDefined();
        expect(now.getAllByText(said.moved)).toHaveLength(2);
        expect(now.getByText("Keep cold")).toBeDefined();
        const marks = (region: HTMLElement): (string | null)[] =>
            [...region.querySelectorAll("[data-card-fact-mark]")].map((node) => node.getAttribute("data-card-fact-mark"));
        expect(marks(proposal.getByRole("region", { name: said.current }))).toStrictEqual(["changed"]);
        expect(marks(proposal.getByRole("region", { name: said.proposed }))).toStrictEqual(["changed", "added"]);
        expect(next.getByText("Above 98%")).toBeDefined();
        expect(next.getByText(said.marks.added)).toBeDefined();
        expect(proposal.getByText("The category page and its products say so.")).toBeDefined();
    });

    it("accepts and dismisses a proposal by its finding", () => {
        drawer();

        const proposal = within(proposalOf("e-peptides"));
        fireEvent.click(proposal.getByRole("button", { name: said.accept }));
        fireEvent.click(proposal.getByRole("button", { name: said.dismiss }));

        expect(held.accepted).toStrictEqual([{ findingId: "f-peptides" }]);
        expect(held.dismissed).toStrictEqual([{ findingId: "f-peptides" }]);
    });

    it("says plainly when the card was edited since, and still lets a person try", () => {
        held.proposals = [peptides({ changed: true })];
        drawer();

        const proposal = within(proposalOf("e-peptides"));
        expect(proposal.getByText(said.editedTitle)).toBeDefined();
        expect(proposal.getByRole("button", { name: said.accept })).toHaveProperty("disabled", false);
    });

    it("explains a refusal because the card moved, with Dismiss at hand", () => {
        held.acceptError = refused("CONFLICT", "the entity's card changed since it was proposed", { reason: "card_changed" });
        drawer();

        const proposal = within(proposalOf("e-peptides"));
        expect(proposal.getByText(said.refusedChanged)).toBeDefined();
        fireEvent.click(proposal.getByRole("button", { name: said.dismiss }));
        expect(held.dismissed).toStrictEqual([{ findingId: "f-peptides" }]);
    });

    it("says a proposal was already settled", () => {
        held.acceptError = refused("CONFLICT", "the proposal is settled");
        drawer();

        expect(within(proposalOf("e-peptides")).getByText(said.refusedSettled)).toBeDefined();
    });

    it("shows the entity on the graph from its name", () => {
        drawer();

        fireEvent.click(within(proposalOf("e-peptides")).getByRole("button", { name: "Peptides" }));
        expect(revealed).toStrictEqual(["e-peptides"]);
    });

    it("counts what waits, and marks the one a person came to review", () => {
        held.proposals = [peptides(), peptides({ findingId: "f-tb", entityId: "e-tb", name: "TB-500" })];
        drawer("e-tb");

        expect(screen.getByText(said.waiting(2, false))).toBeDefined();
        expect(proposalOf("e-tb").className).toContain("border-accent");
        expect(proposalOf("e-peptides").className).not.toContain("border-accent");
    });

    it("says so when nothing waits", () => {
        held.proposals = [];
        drawer();

        expect(screen.getByText(said.noneTitle)).toBeDefined();
    });
});
