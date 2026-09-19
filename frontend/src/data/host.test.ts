import { describe, expect, test } from "vitest";

import { isBrowsable } from "./host.js";

describe("what the host is allowed to open", () => {
    test("accepts the two schemes a WordPress site is served over", () => {
        expect(isBrowsable("https://example.com/shoes/")).toBe(true);
        expect(isBrowsable("http://127.0.0.1:8089/menu/")).toBe(true);
    });

    test("refuses a scheme that is not the web", () => {
        for (const url of [
            "file:///C:/Windows/System32/cmd.exe",
            "javascript:alert(1)",
            "data:text/html,<script>alert(1)</script>",
            "ms-settings:privacy",
            "vbscript:msgbox(1)",
        ]) {
            expect(isBrowsable(url)).toBe(false);
        }
    });

    test("refuses what is not a URL at all", () => {
        expect(isBrowsable("")).toBe(false);
        expect(isBrowsable("/shoes/")).toBe(false);
        expect(isBrowsable("example.com")).toBe(false);
    });
});
