import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import { renderScreen } from "../../../testing/render.js";
import { aProgress } from "../../../testing/sync.js";

const held: { step: string; content: string; asked: unknown[] } = { step: "sync_site", content: "{}", asked: [] };

vi.mock("../../../data/hooks/runs.js", () => ({
    useArtifact: (itemId: string, kind: string) => ({
        isPending: false,
        isError: false,
        error: null,
        data: {
            artifact: {
                id: "a1",
                runId: "r-sync",
                itemId,
                step: held.step,
                kind,
                content: held.content,
                size: held.content.length,
                hash: "",
                purged: false,
                expiresAt: null,
                createdAt: "2026-10-28T09:00:00Z",
            },
        },
    }),
}));

vi.mock("../../../data/hooks/sync.js", () => ({
    useSyncProgress: (siteId: string, runId?: string) => {
        held.asked.push({ siteId, runId });
        return { data: aProgress({ runId: "r-sync", stage: "derive" }), isPending: false };
    },
}));

vi.mock("../../../data/hooks/pages.js", () => ({
    usePageModel: () => ({ data: undefined }),
}));

const { ArtifactPane } = await import("./index.js");

beforeEach(() => {
    held.step = "sync_site";
    held.content = "{}";
    held.asked.length = 0;
});

describe("the sync tab of an item", () => {
    it("shows a site sync's own progress rather than a page's read-back", () => {
        renderScreen(
            <ArtifactPane siteId="s1" itemId="i-sync" runId="r-sync" kind="sync_result" pageId="s1" retentionDays={null} />,
        );

        expect(held.asked).toContainEqual({ siteId: "s1", runId: "r-sync" });
        expect(screen.getByRole("list", { name: copy.sync.stagesLabel })).toBeDefined();
        expect(screen.queryByText(copy.runs.review.publish.wpId)).toBeNull();
    });

    it("keeps a page's read-back after a publish as it was", () => {
        held.step = "sync_back";
        held.content = JSON.stringify({ url: "https://shop.test/about/", status: "publish", wpId: 12, links: 3 });
        renderScreen(
            <ArtifactPane siteId="s1" itemId="i-page" runId="r-gen" kind="sync_result" pageId="p1" retentionDays={null} />,
        );

        expect(held.asked).toEqual([]);
        expect(screen.getByText(copy.runs.review.publish.wpId)).toBeDefined();
    });
});
