import { describe, expect, it } from "vitest";

import { plainText } from "./text.js";

describe("plainText", () => {
    it.each([
        { html: "<p>One</p><p>Two&nbsp;&lt;three&gt;</p>", want: "One Two <three>" },
        { html: "<p>Pull a <strong>shot</strong> &amp; steam</p>", want: "Pull a shot & steam" },
        { html: "", want: "" },
        { html: "&copy; stays", want: "&copy; stays" },
    ])("reads $html as the words a reader sees", ({ html, want }) => {
        expect(plainText(html)).toBe(want);
    });
});
