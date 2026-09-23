import { describe, expect, it } from "vitest";

import { enabledOf, readQuery, wantsNew, writeQuery } from "./params.js";

describe("readQuery", () => {
    it("shows everything by default", () => {
        expect(readQuery(new URLSearchParams(""))).toEqual({ show: "all", id: "" });
    });

    it("keeps the filter and the selection", () => {
        expect(readQuery(new URLSearchParams("show=off&id=s-1"))).toEqual({ show: "off", id: "s-1" });
    });

    it("refuses a filter it does not know", () => {
        expect(readQuery(new URLSearchParams("show=maybe")).show).toBe("all");
    });
});

describe("writeQuery", () => {
    it("writes nothing for the opening state", () => {
        expect(writeQuery({ show: "all", id: "" }).toString()).toBe("");
    });

    it("round-trips", () => {
        const query = { show: "on", id: "s-9" } as const;
        expect(readQuery(writeQuery(query))).toEqual(query);
    });

    it("drops the create flag rather than carrying it", () => {
        expect(writeQuery(readQuery(new URLSearchParams("action=new"))).toString()).toBe("");
    });
});

describe("enabledOf", () => {
    it.each([
        ["all", undefined],
        ["on", true],
        ["off", false],
    ] as const)("filters %s as %s", (show, want) => {
        expect(enabledOf(show)).toBe(want);
    });
});

describe("wantsNew", () => {
    it("opens the empty panel on the palette's query", () => {
        expect(wantsNew(new URLSearchParams("action=new"))).toBe(true);
        expect(wantsNew(new URLSearchParams(""))).toBe(false);
    });
});
