import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ConfirmationCard } from "./card.js";

const said = "The next run will rewrite these pages' own text: /espresso/.";

function card(summary: string | null) {
    return render(
        <ConfirmationCard
            tool="pagetypes_set_scope"
            risk="write"
            status="pending"
            createdAt="2026-10-22T09:00:00Z"
            outcome={null}
            title="Set Body on Hub"
            summary={summary}
            busy={null}
        />,
    );
}

describe("a confirmation card", () => {
    it("says what the call itself asked a person to confirm", () => {
        card(said);

        expect(screen.getByText(said)).toBeDefined();
    });

    it("says nothing more when the call said nothing of its own", () => {
        const { container } = card(null);

        expect(container.textContent).not.toContain(said);
    });
});
