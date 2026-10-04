import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { ImportFinding } from "../../data/types.js";
import { FindingsPanel } from "./findings.js";

function finding(part: Partial<ImportFinding> & { code: string }): ImportFinding {
    return { row: 0, message: "", ...part };
}

describe("the findings of a preview", () => {
    it("places each finding on its sheet and row, or on its sheet alone", () => {
        render(
            <FindingsPanel
                errors={[
                    finding({ sheet: "Catalog", row: 4, code: "scope_clash", message: "Liquid twice under BPC-157" }),
                ]}
                warnings={[
                    finding({ sheet: "Compounds", code: "group_without_page" }),
                    finding({ row: 9, code: "bad_volume" }),
                ]}
                conflicts={[]}
            />,
        );
        expect(screen.getByText("Catalog · row 4")).toBeDefined();
        expect(screen.getByText(copy.imports.findings.scope_clash)).toBeDefined();
        expect(screen.getByText("Liquid twice under BPC-157")).toBeDefined();
        expect(screen.getByText("Compounds")).toBeDefined();
        expect(screen.getByText("row 9")).toBeDefined();
    });

    it("says nothing is wrong when the preview found nothing", () => {
        render(<FindingsPanel errors={[]} warnings={[]} conflicts={[]} />);
        expect(screen.getByText(copy.empty.importFindings)).toBeDefined();
    });
});
