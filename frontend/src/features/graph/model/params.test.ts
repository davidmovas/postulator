import { describe, expect, it } from "vitest";

import { defaultQuery, readQuery, searchOf, writeQuery } from "./params.js";
import type { GraphQuery } from "./params.js";

describe("readQuery and writeQuery", () => {
    it("reads the default from nothing", () => {
        expect(readQuery(new URLSearchParams())).toStrictEqual(defaultQuery);
        expect(defaultQuery).toStrictEqual({ view: "map", lens: "all", presets: [], states: [], isolate: false, proof: false });
    });

    it("round-trips a full query", () => {
        const query: GraphQuery = {
            view: "outline", lens: "noPage", presets: ["product", "untyped"],
            states: ["mismatch", "published"], isolate: true, proof: true,
        };
        const written = writeQuery(query).toString();
        expect(written).toBe("view=outline&lens=noPage&types=product%2Cuntyped&states=mismatch%2Cpublished&isolate=1&proof=1");
        expect(readQuery(new URLSearchParams(written))).toStrictEqual(query);
    });

    it("falls back on values it does not know", () => {
        const read = readQuery(new URLSearchParams("view=foo&lens=bar&types=product,bogus,,hub&states=published,nonsense&isolate=yes&proof=on"));
        expect(read).toStrictEqual({
            view: "map", lens: "all", presets: ["product", "hub"], states: ["published"], isolate: false, proof: false,
        });
    });

    it("ignores the entity kinds an old address still carries", () => {
        expect(readQuery(new URLSearchParams("kinds=product,topic")).presets).toStrictEqual([]);
    });

    it("writes nothing for the default", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
        expect(searchOf({ ...defaultQuery, lens: "orphan" })).toBe("?lens=orphan");
    });
});
