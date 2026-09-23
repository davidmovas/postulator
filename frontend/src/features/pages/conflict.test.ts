import { describe, expect, it } from "vitest";

import { conflictOf } from "./conflict.js";

function rejection(details: Record<string, unknown>, code = "CONFLICT"): Error {
    const thrown = new Error("rejected");
    thrown.cause = { code, message: "page would cannibalize an existing page", details };
    return thrown;
}

describe("conflictOf", () => {
    it("reads the cannibalization evidence a CONFLICT carries", () => {
        const conflict = conflictOf(
            rejection({
                evidence: [
                    { pageId: "p1", path: "/mugs/", reason: "path_conflict", entityId: "" },
                    { pageId: "p2", path: "/cups/", reason: "same_entity_canonical", entityId: "e1" },
                ],
            }),
        );
        expect(conflict).toStrictEqual({
            kind: "cannibalization",
            offenders: [
                { pageId: "p1", path: "/mugs/", reason: "path_conflict", entityId: "" },
                { pageId: "p2", path: "/cups/", reason: "same_entity_canonical", entityId: "e1" },
            ],
        });
    });

    it("reads the descendant refusal a path change raises", () => {
        expect(conflictOf(rejection({ pageId: "p1", descendants: 4 }))).toStrictEqual({
            kind: "descendants",
            descendants: 4,
        });
    });

    it("skips evidence rows that carry neither a page nor a path", () => {
        const conflict = conflictOf(rejection({ evidence: [{ reason: "path_conflict" }, { path: "/a/" }] }));
        expect(conflict).toStrictEqual({
            kind: "cannibalization",
            offenders: [{ pageId: "", path: "/a/", reason: "", entityId: "" }],
        });
    });

    it("ignores every code but CONFLICT", () => {
        expect(conflictOf(rejection({ evidence: [{ pageId: "p1", path: "/a/" }] }, "INVALID"))).toBeNull();
    });

    it("ignores a CONFLICT that explains nothing", () => {
        expect(conflictOf(rejection({}))).toBeNull();
        expect(conflictOf(null)).toBeNull();
        expect(conflictOf(undefined)).toBeNull();
    });
});
