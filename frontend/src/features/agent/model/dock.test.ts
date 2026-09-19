import { describe, expect, it } from "vitest";

import {
    chooseConversation,
    clampWidth,
    defaultDockWidth,
    dockKey,
    maximumDockWidth,
    minimumDockWidth,
    parseChoices,
    parseWidth,
} from "./dock.js";

describe("dockKey", () => {
    it("keys a site by its id and the rest of the app as one global slot", () => {
        expect(dockKey("s1")).toBe("s1");
        expect(dockKey(null)).toBe("global");
    });
});

describe("clampWidth and parseWidth", () => {
    it("keeps the width inside the dock's range", () => {
        expect(clampWidth(100)).toBe(minimumDockWidth);
        expect(clampWidth(10_000)).toBe(maximumDockWidth);
        expect(clampWidth(400)).toBe(400);
    });

    it("reads a stored width and falls back to the default for garbage", () => {
        expect(parseWidth("420")).toBe(420);
        expect(parseWidth("9999")).toBe(maximumDockWidth);
        expect(parseWidth(null)).toBe(defaultDockWidth);
        expect(parseWidth("wide")).toBe(defaultDockWidth);
    });
});

describe("chooseConversation", () => {
    const available = [{ id: "newest" }, { id: "older" }];

    it("prefers the remembered conversation when it is still there", () => {
        expect(chooseConversation("older", available)).toBe("older");
    });

    it("falls back to the newest conversation when the remembered one is gone", () => {
        expect(chooseConversation("deleted", available)).toBe("newest");
        expect(chooseConversation(undefined, available)).toBe("newest");
    });

    it("answers nothing when there is nothing to choose", () => {
        expect(chooseConversation("x", [])).toBeNull();
    });
});

describe("parseChoices", () => {
    it("reads only string to string pairs and ignores anything else", () => {
        expect(parseChoices('{"s1":"c1","global":"c2","bad":3}')).toStrictEqual({ s1: "c1", global: "c2" });
        expect(parseChoices("[1,2]")).toStrictEqual({});
        expect(parseChoices("{")).toStrictEqual({});
        expect(parseChoices(null)).toStrictEqual({});
    });
});
