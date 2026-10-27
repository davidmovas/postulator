import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import { renderScreen } from "../../../testing/render.js";

vi.mock("../../../data/hooks/tools.js", () => ({
    useTools: () => ({
        data: {
            tools: [{ name: "pages_manage", description: "Plan, change or delete pages.", risk: "dangerous", schema: null }],
        },
    }),
}));

vi.mock("../../../data/hooks/templates.js", () => ({
    useTemplate: () => ({ data: undefined }),
}));

vi.mock("../../../data/hooks/findings.js", () => ({
    useFinding: () => ({ data: undefined }),
}));

vi.mock("./ref.js", () => ({
    RefLabel: ({ id }: { id: string }) => <span>{id}</span>,
}));

const { ActionCard } = await import("./action-card.js");

function card(args: Record<string, unknown>, risk: string | null) {
    return renderScreen(
        <ActionCard
            tool="pages_manage"
            args={args}
            summary="Change the page in the page map"
            risk={risk}
            status="pending"
            createdAt="2026-10-27T09:00:00Z"
            outcome={null}
            busy={null}
        />,
    );
}

describe("the risk an approval card is badged with", () => {
    it("badges an update by the risk its own op carries, a write", () => {
        card({ op: "update", pageId: "p1", title: "Mugs" }, "write");

        expect(screen.getByText(copy.agent.card.riskWrite)).toBeDefined();
        expect(screen.queryByText(copy.agent.card.dangerousWarn)).toBeNull();
    });

    it("badges a delete as dangerous and warns it cannot be undone", () => {
        card({ op: "delete", pageId: "p1" }, "dangerous");

        expect(screen.getByText(copy.agent.card.riskDangerous)).toBeDefined();
        expect(screen.getByText(copy.agent.card.dangerousWarn)).toBeDefined();
    });

    it("falls back to the tool's own risk for a card stored before its op carried one", () => {
        card({ op: "update", pageId: "p1" }, "");

        expect(screen.getByText(copy.agent.card.riskDangerous)).toBeDefined();
    });
});
