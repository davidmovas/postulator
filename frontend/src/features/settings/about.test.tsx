import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";

const said = copy.settings.about.support;

interface Held {
    saved: { path: string; bytes: number } | null;
    asked: { siteId: string; siteName: string }[];
    error: unknown;
    sites: { id: string; name: string }[];
}

const held: Held = { saved: null, asked: [], error: null, sites: [] };

vi.mock("../../data/hooks/health.js", () => ({
    useBuildInfo: () => ({
        data: { version: "2.7.0", commit: "abc1234", buildDate: "" },
        isFetching: false,
        refetch: () => undefined,
    }),
}));

vi.mock("../../data/hooks/sites.js", () => ({
    useSites: () => ({ data: { pages: [{ items: held.sites, hasMore: false }] }, isPending: false }),
}));

vi.mock("../../data/hooks/support.js", async (original) => ({
    ...(await original<typeof import("../../data/hooks/support.js")>()),
    useSaveBundle: () => ({
        mutate: (request: { siteId: string; siteName: string }) => {
            held.asked.push(request);
        },
        reset: () => undefined,
        isPending: false,
        data: held.saved,
        error: held.error,
    }),
}));

const { AboutScreen } = await import("./about.js");

beforeEach(() => {
    held.saved = null;
    held.asked = [];
    held.error = null;
    held.sites = [{ id: "s1", name: "Clay and Kiln" }];
});

describe("the support bundle in About", () => {
    it("says what the bundle holds and that it holds no secret and no page text", () => {
        renderScreen(<AboutScreen />);

        expect(screen.getByText(said.holds)).toBeDefined();
        expect(said.holds).toMatch(/no key, no password and no text of any page/);
    });

    it("saves a bundle of the site", () => {
        renderScreen(<AboutScreen />);

        fireEvent.click(screen.getByRole("button", { name: said.save }));

        expect(held.asked).toStrictEqual([{ siteId: "s1", siteName: "Clay and Kiln" }]);
    });

    it("says where the bundle was written", () => {
        held.saved = { path: "C:\\Users\\me\\support.json", bytes: 20480 };
        renderScreen(<AboutScreen />);

        expect(screen.getByText(said.saved("C:\\Users\\me\\support.json", "20.0 KB"))).toBeDefined();
    });

    it("says why a bundle could not be written", () => {
        held.error = Object.assign(new Error("refused"), {
            cause: { code: "EXTERNAL", message: "the support bundle could not be written to C:\\locked.json" },
        });
        renderScreen(<AboutScreen />);

        expect(screen.getByText("the support bundle could not be written to C:\\locked.json")).toBeDefined();
    });

    it("asks for a site first when there is none", () => {
        held.sites = [];
        renderScreen(<AboutScreen />);

        expect(screen.getByText(said.noSites)).toBeDefined();
        expect(screen.queryByRole("button", { name: said.save })).toBeNull();
    });
});
