import { describe, expect, it } from "vitest";

import { pathFilter, readQuery, writeQuery } from "./params.js";

describe("readQuery", () => {
    it("opens on the site tab", () => {
        expect(readQuery(new URLSearchParams(""))).toEqual({ tab: "site", runId: "", pageId: "", prefix: "" });
    });

    it("keeps the run a reload was reading", () => {
        expect(readQuery(new URLSearchParams("tab=runs&run=r-1")).runId).toBe("r-1");
    });

    it("refuses a tab it does not know", () => {
        expect(readQuery(new URLSearchParams("tab=dance")).tab).toBe("site");
    });
});

describe("writeQuery", () => {
    it("writes nothing for the opening state", () => {
        expect(writeQuery({ tab: "site", runId: "", pageId: "", prefix: "" }).toString()).toBe("");
    });

    it("leaves a run out of the pages tab and a page out of the runs tab", () => {
        expect(writeQuery({ tab: "pages", runId: "r-1", pageId: "p-1", prefix: "/a/" }).toString()).toBe(
            "tab=pages&prefix=%2Fa%2F&page=p-1",
        );
        expect(writeQuery({ tab: "runs", runId: "r-1", pageId: "p-1", prefix: "/a/" }).toString()).toBe(
            "tab=runs&run=r-1",
        );
    });

    it("round-trips the runs tab", () => {
        const query = { tab: "runs", runId: "r-1", pageId: "", prefix: "" } as const;
        expect(readQuery(writeQuery(query))).toEqual(query);
    });
});

describe("pathFilter", () => {
    it.each([
        ["guides", "/guides"],
        ["/guides/", "/guides/"],
        ["  mugs ", "/mugs"],
        ["", ""],
        ["   ", ""],
    ])("reads %s as %s", (typed, want) => {
        expect(pathFilter(typed)).toBe(want);
    });
});
