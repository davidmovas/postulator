import { describe, expect, it } from "vitest";

import type { SettingDescriptor } from "../../../data/types.js";
import { boundNumber, boundText, checkSetting, kindOf, parseDuration, renderValue, sameValue } from "./value.js";

function descriptor(over: Partial<SettingDescriptor>): SettingDescriptor {
    return { key: "runs.workers", group: "runs", type: "int", default: 2, ...over } as SettingDescriptor;
}

describe("parseDuration", () => {
    it.each([
        ["5s", 5_000_000_000],
        ["2m30s", 150_000_000_000],
        ["1h0m0s", 3_600_000_000_000],
        ["500ms", 500_000_000],
        ["1.5h", 5_400_000_000_000],
        ["100ns", 100],
        ["0", 0],
        ["-3s", -3_000_000_000],
    ])("reads %s", (text, want) => {
        expect(parseDuration(text)).toBe(want);
    });

    it.each(["", "   ", "30", "30x", "h", "s5", "1h2"])("refuses %s", (text) => {
        expect(parseDuration(text)).toBeNull();
    });
});

describe("boundNumber and boundText", () => {
    it("narrows only what it can use", () => {
        expect(boundNumber(8)).toBe(8);
        expect(boundNumber("8")).toBeNull();
        expect(boundNumber(undefined)).toBeNull();
        expect(boundNumber(Number.NaN)).toBeNull();
        expect(boundText("5s")).toBe("5s");
        expect(boundText(5)).toBeNull();
        expect(boundText("")).toBeNull();
    });
});

describe("kindOf", () => {
    it.each([
        ["bool", "bool"],
        ["int", "int"],
        ["enum", "enum"],
        ["duration", "duration"],
        ["string", "string"],
        ["something else", "string"],
    ])("reads %s as %s", (type, want) => {
        expect(kindOf(descriptor({ type }))).toBe(want);
    });
});

describe("sameValue", () => {
    it("compares durations by length, not by text", () => {
        expect(sameValue("duration", "60s", "1m")).toBe(true);
        expect(sameValue("duration", "1m0s", "1m")).toBe(true);
        expect(sameValue("duration", "30s", "1m")).toBe(false);
        expect(sameValue("duration", "nonsense", "nonsense")).toBe(true);
        expect(sameValue("int", 8, 8)).toBe(true);
        expect(sameValue("int", 8, 9)).toBe(false);
        expect(sameValue("string", "", "")).toBe(true);
    });
});

describe("renderValue", () => {
    it.each([
        ["bool", true, "true"],
        ["bool", false, "false"],
        ["int", 8, "8"],
        ["int", undefined, ""],
        ["string", "keep", "keep"],
        ["duration", "30s", "30s"],
        ["string", 4, ""],
    ] as const)("renders %s %s", (kind, raw, want) => {
        expect(renderValue(kind, raw)).toBe(want);
    });
});

describe("checkSetting", () => {
    it("refuses an integer outside its bounds", () => {
        const held = descriptor({ type: "int", min: 1, max: 16 });
        expect(checkSetting(held, "1")).toBeNull();
        expect(checkSetting(held, "16")).toBeNull();
        expect(checkSetting(held, "0")).toEqual({ kind: "range", min: "1", max: "16" });
        expect(checkSetting(held, "99")).toEqual({ kind: "range", min: "1", max: "16" });
        expect(checkSetting(held, "eight")).toEqual({ kind: "shape" });
    });

    it("refuses a duration outside its bounds", () => {
        const held = descriptor({ key: "runs.deadline", type: "duration", min: "5m", max: "168h" });
        expect(checkSetting(held, "30m")).toBeNull();
        expect(checkSetting(held, "1m")).toEqual({ kind: "range", min: "5m", max: "168h" });
        expect(checkSetting(held, "300s")).toBeNull();
        expect(checkSetting(held, "banana")).toEqual({ kind: "shape" });
    });

    it("refuses an empty value only where the declaration says so", () => {
        const required = descriptor({ key: "images.openaiModel", type: "string", nonEmpty: true });
        expect(checkSetting(required, "  ")).toEqual({ kind: "empty" });
        expect(checkSetting(required, "a-model")).toBeNull();
        expect(checkSetting(descriptor({ key: "wp.proxyUrl", type: "string" }), "")).toBeNull();
    });
});
