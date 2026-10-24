import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { PageLink } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { PageLinks } from "./links.js";

function link(overrides: Partial<PageLink>): PageLink {
    return {
        id: "l1",
        siteId: "s1",
        fromPageId: "p1",
        toPageId: "p2",
        toUrl: "/peptides/",
        anchorText: "Peptides",
        type: "breadcrumb",
        region: "breadcrumb",
        source: "render",
        position: null,
        ours: false,
        observedAt: "2026-10-18T07:00:00Z",
        ...overrides,
    };
}

describe("the drawer's links tab", () => {
    it("says each link's type and where it was read, in words", () => {
        const { container } = renderScreen(
            <PageLinks
                links={[
                    link({ id: "l1" }),
                    link({ id: "l2", toUrl: "/bpc-157/", type: "contextual", region: "content", source: "raw", ours: true }),
                ]}
                siteId="s1"
                search=""
            />,
        );

        expect(screen.getByText("Breadcrumb · in the render")).toBeDefined();
        expect(screen.getByText("Contextual · in the body")).toBeDefined();
        expect(screen.getByLabelText(copy.pages.owners.ours)).toBeDefined();
        expect(screen.getByLabelText(copy.pages.owners.theirs)).toBeDefined();
        expect(screen.getByText(copy.pages.detail.linksSummary(2, 1))).toBeDefined();
        expect(container.innerHTML).not.toContain('title="true"');
        expect(container.innerHTML).not.toContain('title="false"');
    });
});
