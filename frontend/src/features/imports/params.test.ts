import { describe, expect, it } from "vitest";

import { importSteps, readQuery, reachable, stepAfter, stepBefore, stepIndex, writeQuery } from "./params.js";

function read(search: string) {
    return readQuery(new URLSearchParams(search));
}

describe("readQuery", () => {
    it("lands on the file step with nothing chosen", () => {
        expect(read("")).toEqual({ tab: "import", step: "file", path: "", mappingId: "" });
    });

    it("keeps the step a reload was on when the file came with it", () => {
        expect(read("step=preview&path=C:%5Csheets%5Cmugs.csv")).toEqual({
            tab: "import",
            step: "preview",
            path: "C:\\sheets\\mugs.csv",
            mappingId: "",
        });
    });

    it("refuses a step that has no file behind it", () => {
        expect(read("step=apply").step).toBe("file");
    });

    it("refuses a tab and a step it does not know", () => {
        expect(read("tab=dance&step=fly&path=x")).toEqual({
            tab: "import",
            step: "file",
            path: "x",
            mappingId: "",
        });
    });

    it("carries the saved mapping it was opened with", () => {
        expect(read("path=x&mapping=m-1").mappingId).toBe("m-1");
    });
});

describe("writeQuery", () => {
    it("writes nothing for the opening state", () => {
        expect(writeQuery({ tab: "import", step: "file", path: "", mappingId: "" }).toString()).toBe("");
    });

    it("round-trips a chosen file on a later step", () => {
        const query = { tab: "import", step: "columns", path: "C:\\a b\\x.csv", mappingId: "m-1" } as const;
        expect(readQuery(writeQuery(query))).toEqual(query);
    });

    it("drops a step that has no file", () => {
        expect(writeQuery({ tab: "import", step: "preview", path: "", mappingId: "" }).toString()).toBe("");
    });

    it("keeps the export tab", () => {
        expect(writeQuery({ tab: "export", step: "file", path: "", mappingId: "" }).toString()).toBe("tab=export");
    });
});

describe("step order", () => {
    it.each([
        ["file", 0],
        ["columns", 1],
        ["preview", 2],
        ["apply", 3],
    ] as const)("puts %s at %i", (step, at) => {
        expect(stepIndex(step)).toBe(at);
    });

    it("walks forward and back without leaving the list", () => {
        expect(stepAfter("file")).toBe("columns");
        expect(stepAfter("apply")).toBe("apply");
        expect(stepBefore("columns")).toBe("file");
        expect(stepBefore("file")).toBe("file");
    });

    it("opens only the file step before a file is chosen", () => {
        for (const step of importSteps) {
            expect(reachable(step, "")).toBe(step === "file");
            expect(reachable(step, "x.csv")).toBe(true);
        }
    });
});
