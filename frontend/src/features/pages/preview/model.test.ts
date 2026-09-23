import { describe, expect, it } from "vitest";

import { editorPreviewUrl, expiresInMinutes, frameScale, previewState, viewports } from "./model.js";

const withPreview = { installed: true, capabilities: ["bulk", "raw", "preview"] };
const olderPlugin = { installed: true, capabilities: ["bulk", "raw"] };
const noPlugin = { installed: false, capabilities: [] };

describe("previewState", () => {
    it("says a planned page or one without a WordPress id is not on the site", () => {
        expect(previewState({ status: "planned", wpId: null }, withPreview)).toBe("not-on-site");
        expect(previewState({ status: "exists", wpId: null }, withPreview)).toBe("not-on-site");
    });

    it("says an archived page is gone even when it kept its id", () => {
        expect(previewState({ status: "archived", wpId: 7 }, withPreview)).toBe("archived");
    });

    it("opens a published page by its address whatever the plugin", () => {
        expect(previewState({ status: "published", wpId: 7 }, noPlugin)).toBe("public");
    });

    it("previews a draft through the plugin and says what is missing otherwise", () => {
        expect(previewState({ status: "exists", wpId: 7 }, withPreview)).toBe("draft");
        expect(previewState({ status: "exists", wpId: 7 }, olderPlugin)).toBe("draft-plugin-outdated");
        expect(previewState({ status: "exists", wpId: 7 }, noPlugin)).toBe("draft-needs-plugin");
    });

    it("lets the site decide while its plugin state is unknown", () => {
        expect(previewState({ status: "exists", wpId: 7 }, null)).toBe("draft");
    });
});

describe("expiresInMinutes", () => {
    const now = Date.parse("2026-09-19T10:00:00Z");

    it("counts whole minutes down and never below zero", () => {
        expect(expiresInMinutes("2026-09-19T10:58:30Z", now)).toBe(58);
        expect(expiresInMinutes("2026-09-19T10:00:20Z", now)).toBe(0);
        expect(expiresInMinutes("2026-09-19T09:00:00Z", now)).toBe(0);
    });

    it("answers nothing for a link that does not expire", () => {
        expect(expiresInMinutes(null, now)).toBeNull();
        expect(expiresInMinutes("soon", now)).toBeNull();
    });
});

describe("frameScale", () => {
    it("shrinks a wide viewport to fit and never enlarges a narrow one", () => {
        expect(frameScale(640, viewports.desktop)).toBe(0.5);
        expect(frameScale(1000, viewports.phone)).toBe(1);
        expect(frameScale(0, viewports.desktop)).toBe(1);
    });
});

describe("editorPreviewUrl", () => {
    it("points WordPress at the draft by its id so a logged-in editor sees it", () => {
        expect(editorPreviewUrl("https://clay.example.com", "page", 42)).toBe("https://clay.example.com/?page_id=42&preview=true");
        expect(editorPreviewUrl("https://clay.example.com/", "post", 7)).toBe("https://clay.example.com/?p=7&preview=true");
    });

    it("answers nothing without an id or an address", () => {
        expect(editorPreviewUrl("https://clay.example.com", "page", null)).toBeNull();
        expect(editorPreviewUrl("", "page", 42)).toBeNull();
    });
});
