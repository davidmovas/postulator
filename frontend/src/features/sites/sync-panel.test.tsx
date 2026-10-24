import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";
import { aSite, siteSettings } from "../../testing/sites.js";

const sent: unknown[] = [];

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncSite: () => ({
        mutate: (request: unknown) => {
            sent.push(request);
        },
        isPending: false,
    }),
}));

const { SiteSyncPanel } = await import("./sync-panel.js");

const said = copy.sites.sync;

beforeEach(() => {
    sent.length = 0;
});

describe("the site's sync options", () => {
    it("syncs what changed since the last walk when no option is ticked", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: false, renderRescan: false }]);
    });

    it("asks for a full walk", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByLabelText(said.full));
        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: true, renderRescan: false }]);
    });

    it("asks for every page to be rendered again, with a full walk or without", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByLabelText(said.rescan));
        fireEvent.click(screen.getByRole("button", { name: said.start }));
        fireEvent.click(screen.getByLabelText(said.full));
        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([
            { siteId: "s1", full: false, renderRescan: true },
            { siteId: "s1", full: true, renderRescan: true },
        ]);
    });

    it("says when the site was last walked and never for a site never walked", () => {
        const never = siteSettings({
            observation: { lastStartedAt: null, lastFinishedAt: null, lastFullWalkAt: null, lastModifiedSeen: null, sinceFullWalk: 0 },
        });
        renderScreen(<SiteSyncPanel site={aSite({ settings: never })} />);

        expect(screen.getByText(said.lastWalk)).toBeDefined();
        expect(screen.getAllByText("never")).toHaveLength(2);
    });
});
