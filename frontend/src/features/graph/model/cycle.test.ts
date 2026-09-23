import { describe, expect, it } from "vitest";

import { cycleOf } from "./cycle.js";

function rejection(details: Record<string, unknown>, code = "INVALID"): Error {
    const thrown = new Error("rejected");
    thrown.cause = { code, message: "parent edges form a cycle", details };
    return thrown;
}

describe("cycleOf", () => {
    it("reads the entity ids of the cycle an INVALID carries", () => {
        expect(cycleOf(rejection({ cycle: ["a", "b", "a"] }))).toStrictEqual(["a", "b", "a"]);
    });

    it("answers nothing for another code or a missing cycle", () => {
        expect(cycleOf(rejection({ cycle: ["a", "b", "a"] }, "CONFLICT"))).toBeNull();
        expect(cycleOf(rejection({ field: "kind" }))).toBeNull();
        expect(cycleOf(rejection({ cycle: [] }))).toBeNull();
        expect(cycleOf(null)).toBeNull();
    });

    it("keeps only the string entries", () => {
        expect(cycleOf(rejection({ cycle: ["a", 3, null, "b"] }))).toStrictEqual(["a", "b"]);
    });
});
