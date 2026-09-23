import { describe, expect, it } from "vitest";

import {
    chooseConversation,
    clampWidth,
    dockKey,
    globalDockKey,
    maximumDockWidth,
    minimumDockWidth,
    narrowDockWidth,
    parseChoices,
    parseCustom,
    parseMode,
    widthOf,
} from "./state.js";

describe("the dock width", () => {
    it("narrow is the contract's 392 pixels on a design-sized window", () => {
        expect(widthOf("narrow", null, 1280)).toBe(narrowDockWidth);
    });

    it("wide is half the window", () => {
        expect(widthOf("wide", null, 1280)).toBe(640);
    });

    it("leaves the main column usable on the smallest window", () => {
        expect(widthOf("wide", null, 960)).toBe(480);
        expect(widthOf("narrow", null, 960)).toBe(narrowDockWidth);
        expect(widthOf("wide", 900, 960)).toBe(600);
    });

    it("a dragged width wins over the mode and stays inside the bounds", () => {
        expect(widthOf("narrow", 500, 1280)).toBe(500);
        expect(widthOf("wide", 10, 1280)).toBe(minimumDockWidth);
        expect(widthOf("narrow", 4000, 4000)).toBe(maximumDockWidth);
    });

    it("clamping never returns less than the minimum even on a tiny window", () => {
        expect(clampWidth(400, 500)).toBe(minimumDockWidth);
    });
});

describe("what the dock remembers", () => {
    it("keys a choice by site, or globally when there is no site", () => {
        expect(dockKey("s1")).toBe("s1");
        expect(dockKey(null)).toBe(globalDockKey);
        expect(dockKey("")).toBe(globalDockKey);
    });

    it("falls back to the newest conversation when the remembered one is gone", () => {
        const available = [{ id: "c2" }, { id: "c1" }];
        expect(chooseConversation("c1", available)).toBe("c1");
        expect(chooseConversation("c9", available)).toBe("c2");
        expect(chooseConversation(undefined, [])).toBeNull();
    });

    it("reads back only string choices and survives rubbish", () => {
        expect(parseChoices('{"s1":"c1","s2":3}')).toStrictEqual({ s1: "c1" });
        expect(parseChoices("[1,2]")).toStrictEqual({});
        expect(parseChoices("not json")).toStrictEqual({});
        expect(parseChoices(null)).toStrictEqual({});
    });

    it("reads back the mode and the dragged width", () => {
        expect(parseMode("wide")).toBe("wide");
        expect(parseMode(null)).toBe("narrow");
        expect(parseMode("anything")).toBe("narrow");
        expect(parseCustom("480")).toBe(480);
        expect(parseCustom("")).toBeNull();
        expect(parseCustom(null)).toBeNull();
    });
});
