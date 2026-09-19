import { describe, expect, it } from "vitest";

import { defaultQuery, readQuery, searchOf, writeQuery } from "./params.js";

describe("readQuery and writeQuery", () => {
    it("reads the default from nothing", () => {
        expect(readQuery(new URLSearchParams())).toStrictEqual(defaultQuery);
        expect(defaultQuery).toStrictEqual({ view: "map", lens: "all", kinds: [], isolate: false, proof: false });
    });

    it("round-trips a full query", () => {
        const query = { view: "outline" as const, lens: "noPage" as const, kinds: ["product", "topic"], isolate: true, proof: true };
        const written = writeQuery(query).toString();
        expect(written).toBe("view=outline&lens=noPage&kinds=product%2Ctopic&isolate=1&proof=1");
        expect(readQuery(new URLSearchParams(written))).toStrictEqual(query);
    });

    it("falls back on values it does not know", () => {
        const read = readQuery(new URLSearchParams("view=foo&lens=bar&kinds=product,bogus,,topic&isolate=yes&proof=on"));
        expect(read).toStrictEqual({ view: "map", lens: "all", kinds: ["product", "topic"], isolate: false, proof: false });
    });

    it("writes nothing for the default", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
        expect(searchOf({ ...defaultQuery, lens: "orphan" })).toBe("?lens=orphan");
    });
});
