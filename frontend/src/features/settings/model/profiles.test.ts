import { describe, expect, it } from "vitest";

import { modelRoles } from "../../../generated/vocab.js";
import { pickableRoles, profileRows, refOf, refText } from "./profiles.js";

const roles = ["writer", "chat", "image"] as const;

describe("profileRows", () => {
    it("reads a chosen model as chosen here", () => {
        const rows = profileRows(roles, [
            { role: "writer", global: { provider: "openai", model: "a" }, effective: { provider: "openai", model: "a" } },
        ]);
        expect(rows[0]).toEqual({
            role: "writer",
            chosen: { provider: "openai", model: "a" },
            effective: { provider: "openai", model: "a" },
            source: "global",
        });
    });

    it("reads an effective model with no choice as the shipped default", () => {
        const rows = profileRows(roles, [{ role: "chat", effective: { provider: "openai", model: "b" } }]);
        expect(rows[1]?.source).toBe("seeded");
        expect(rows[1]?.chosen).toBeNull();
    });

    it("reads an empty ref as nothing set", () => {
        const rows = profileRows(roles, [{ role: "image", global: { provider: "", model: "" }, effective: null }]);
        expect(rows[2]?.source).toBe("none");
        expect(rows[2]?.effective).toBeNull();
    });

    it("keeps a row for a role the answer never names", () => {
        const rows = profileRows(roles, []);
        expect(rows.map((row) => row.role)).toEqual(["writer", "chat", "image"]);
        expect(rows.every((row) => row.source === "none")).toBe(true);
    });
});

describe("refText and refOf", () => {
    it("round trips a reference", () => {
        expect(refText({ provider: "openai", model: "gpt-5.6-luna" })).toBe("openai/gpt-5.6-luna");
        expect(refOf("openai/gpt-5.6-luna")).toEqual({ provider: "openai", model: "gpt-5.6-luna" });
        expect(refText(null)).toBe("");
    });

    it.each(["", "openai", "/model", "openai/"])("refuses %s", (text) => {
        expect(refOf(text)).toBeNull();
    });
});

describe("pickableRoles", () => {
    it("offers every role but the one the Images setting answers", () => {
        expect(pickableRoles).not.toContain("image");
        expect(pickableRoles).toStrictEqual(modelRoles.filter((role) => role !== "image"));
    });
});
