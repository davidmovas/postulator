import { describe, expect, it } from "vitest";

import type { ImportMapping, ImportOptions, ImportSheet } from "../../data/types.js";
import {
    activateSheet,
    adoptSaved,
    adoptSavedOn,
    assignColumn,
    chooseSheet,
    chosenRows,
    columnsNotice,
    headerless,
    inUse,
    openWorkbook,
    requestOf,
    rowsOf,
    savedNames,
    setSheetOptions,
    settingsOf,
    sheetsOf,
} from "./workbook.js";
import type { Workbook } from "./workbook.js";

function mapping(columns: Record<string, string>, options: ImportOptions = {}, id?: string): ImportMapping {
    return { id, columns, options, createdAt: null, updatedAt: null };
}

function sheet(name: string, rows: number, headers: string[], detected: ImportMapping): ImportSheet {
    return { name, rows, headers, detected };
}

const catalog = sheet(
    "Catalog",
    3,
    ["Root Entity", "Category", "URL", "Title"],
    mapping(
        { path: "URL", title: "Title" },
        { sheets: ["Catalog"], rowType: "pages" as ImportOptions["rowType"], levelColumns: ["Root Entity"] },
    ),
);
const peptides = sheet(
    "Peptides",
    2,
    ["Entity", "Keywords"],
    mapping({ entity: "Entity", keywords: "Keywords" }, { sheets: ["Peptides"] }),
);
const empty = sheet("Readme", 0, [], mapping({}, { sheets: ["Readme"] }));
const forms = sheet("Forms", 4, ["Name", "Notes"], mapping({}, { sheets: ["Forms"], noteColumns: ["Notes"] }));

function book(): Workbook {
    return openWorkbook([catalog, peptides, empty, forms]);
}

describe("opening a workbook", () => {
    it("chooses every sheet that has rows, in the order of the workbook, and opens the first", () => {
        const opened = book();
        expect(opened.chosen).toEqual(["Catalog", "Peptides", "Forms"]);
        expect(opened.active).toBe("Catalog");
    });

    it("gives each sheet the columns, groups, notes and row type detected on it", () => {
        const opened = book();
        expect(settingsOf(opened, "Catalog")).toEqual({
            columns: { path: "URL", title: "Title" },
            options: { sheets: ["Catalog"], rowType: "pages", levelColumns: ["Root Entity"] },
            mappingId: "",
        });
        expect(settingsOf(opened, "Forms").options.noteColumns).toEqual(["Notes"]);
    });

    it("falls back to the first sheet when no sheet has rows", () => {
        const opened = openWorkbook([empty, sheet("Blank", 0, [], mapping({}))]);
        expect(opened.chosen).toEqual(["Readme"]);
        expect(opened.active).toBe("Readme");
    });

    it("reads a csv as its one unnamed sheet", () => {
        const opened = openWorkbook([sheet("", 5, ["URL"], mapping({ path: "URL" }))]);
        expect(opened.chosen).toEqual([""]);
        expect(opened.active).toBe("");
    });
});

describe("choosing sheets", () => {
    it("keeps every sheet's settings when a sheet is turned off and on again", () => {
        let held = activateSheet(book(), "Peptides");
        held = assignColumn(held, "Keywords", "anchors");
        held = chooseSheet(held, "Peptides", false);
        held = chooseSheet(held, "Catalog", false);
        held = chooseSheet(held, "Peptides", true);
        held = chooseSheet(held, "Catalog", true);
        expect(settingsOf(held, "Peptides").columns).toEqual({ entity: "Entity", anchors: "Keywords" });
        expect(settingsOf(held, "Catalog")).toEqual(settingsOf(book(), "Catalog"));
    });

    it("keeps the chosen sheets in the order of the workbook whatever order they were turned on in", () => {
        let held = chooseSheet(book(), "Catalog", false);
        held = chooseSheet(held, "Readme", true);
        held = chooseSheet(held, "Catalog", true);
        expect(held.chosen).toEqual(["Catalog", "Peptides", "Readme", "Forms"]);
    });

    it.each([
        { name: "a sheet turned on becomes the open tab", from: "Catalog", sheet: "Readme", on: true, want: "Readme" },
        {
            name: "the open tab moves to the first sheet left when it is turned off",
            from: "Catalog",
            sheet: "Catalog",
            on: false,
            want: "Peptides",
        },
        {
            name: "another sheet turned off leaves the open tab alone",
            from: "Forms",
            sheet: "Peptides",
            on: false,
            want: "Forms",
        },
    ])("$name", ({ from, sheet: name, on, want }) => {
        expect(chooseSheet(activateSheet(book(), from), name, on).active).toBe(want);
    });

    it("leaves no tab open once every sheet is off", () => {
        let held = book();
        for (const name of ["Catalog", "Peptides", "Forms"]) {
            held = chooseSheet(held, name, false);
        }
        expect(held.chosen).toEqual([]);
        expect(held.active).toBe("");
    });

    it("ignores a sheet the workbook does not have", () => {
        const opened = book();
        expect(chooseSheet(opened, "Ghost", true)).toBe(opened);
    });

    it("opens only a chosen sheet's tab", () => {
        expect(activateSheet(book(), "Readme").active).toBe("Catalog");
        expect(activateSheet(book(), "Forms").active).toBe("Forms");
    });
});

describe("editing the open sheet", () => {
    it("changes the columns and options of the open tab and no other", () => {
        let held = activateSheet(book(), "Forms");
        held = assignColumn(held, "Name", "entity");
        held = setSheetOptions(held, { ...settingsOf(held).options, rowType: "products" as ImportOptions["rowType"] });
        expect(settingsOf(held, "Forms").columns).toEqual({ entity: "Name" });
        expect(settingsOf(held, "Forms").options.rowType).toBe("products");
        expect(settingsOf(held, "Catalog")).toEqual(settingsOf(book(), "Catalog"));
    });

    it("reads a sheet without a header row from scratch and goes back to what was detected", () => {
        const without = headerless(book(), true);
        expect(settingsOf(without, "Catalog")).toEqual({
            columns: {},
            options: {
                sheets: ["Catalog"],
                rowType: "pages",
                noHeader: true,
                indentColumns: [],
                levelColumns: [],
                noteColumns: [],
            },
            mappingId: "",
        });
        expect(settingsOf(headerless(without, false), "Catalog")).toEqual(settingsOf(book(), "Catalog"));
    });
});

describe("a saved mapping", () => {
    const saved = mapping(
        { path: "Address", h1: "Heading" },
        { sheets: ["Peptides"], pathPrefixStrip: "https://shop.example" },
        "m-1",
    );

    it("goes to the sheet it was saved for and opens it", () => {
        const held = adoptSaved(book(), saved);
        expect(held.active).toBe("Peptides");
        expect(settingsOf(held, "Peptides")).toEqual({
            columns: { path: "Address", h1: "Heading" },
            options: { sheets: ["Peptides"], pathPrefixStrip: "https://shop.example" },
            mappingId: "m-1",
        });
        expect(settingsOf(held, "Catalog")).toEqual(settingsOf(book(), "Catalog"));
    });

    it("chooses the sheet it was saved for when that sheet was off", () => {
        const held = adoptSaved(chooseSheet(book(), "Peptides", false), saved);
        expect(held.chosen).toEqual(["Catalog", "Peptides", "Forms"]);
    });

    it("goes to the open sheet when the workbook has no sheet of its name", () => {
        const held = adoptSaved(activateSheet(book(), "Forms"), { ...saved, options: { sheets: ["Elsewhere"] } });
        expect(settingsOf(held, "Forms").columns).toEqual({ path: "Address", h1: "Heading" });
        expect(settingsOf(held, "Forms").options.sheets).toEqual(["Forms"]);
    });

    it("goes to the tab it was picked on", () => {
        const held = adoptSavedOn(book(), saved, "Catalog");
        expect(settingsOf(held, "Catalog").mappingId).toBe("m-1");
        expect(settingsOf(held, "Catalog").options.sheets).toEqual(["Catalog"]);
        expect(settingsOf(held, "Peptides")).toEqual(settingsOf(book(), "Peptides"));
    });

    it("is reported in use only while its sheet is chosen", () => {
        const held = adoptSaved(book(), saved);
        expect(inUse(held)).toEqual(["m-1"]);
        expect(inUse(chooseSheet(held, "Peptides", false))).toEqual([]);
    });

    it("keeps of its group columns only the root headers detected on the sheet", () => {
        const grouped = mapping({ path: "URL" }, { levelColumns: ["Root Entity", "Category"] }, "m-2");
        const held = adoptSavedOn(book(), grouped, "Catalog");
        expect(settingsOf(held, "Catalog").options.levelColumns).toEqual(["Root Entity"]);
    });

    it("maps nothing when a category column was all it grouped by", () => {
        const grouped = mapping({}, { levelColumns: ["Category"] }, "m-3");
        const held = adoptSavedOn(book(), grouped, "Catalog");
        expect(settingsOf(held, "Catalog").options.levelColumns).toEqual([]);
        expect(columnsNotice(held).kind).toBe("needsTarget");
    });
});

describe("the request a workbook sends", () => {
    it("sends one chosen sheet as the one mapping of the file", () => {
        let held = book();
        for (const name of ["Catalog", "Forms"]) {
            held = chooseSheet(held, name, false);
        }
        expect(requestOf(held)).toEqual({
            mapping: {
                columns: { entity: "Entity", keywords: "Keywords" },
                options: { sheets: ["Peptides"] },
                createdAt: null,
                updatedAt: null,
            },
        });
    });

    it("sends several sheets each with its own mapping, in workbook order, and no mapping for the file", () => {
        let held = chooseSheet(book(), "Catalog", false);
        held = chooseSheet(held, "Catalog", true);
        held = assignColumn(activateSheet(held, "Forms"), "Name", "entity");
        const request = requestOf(held);
        expect(request?.mapping).toEqual({ columns: null, options: {}, createdAt: null, updatedAt: null });
        expect(request?.sheets?.map((each) => each.sheet)).toEqual(["Catalog", "Peptides", "Forms"]);
        expect(request?.sheets?.[2]?.mapping).toEqual({
            columns: { entity: "Name" },
            options: { sheets: ["Forms"], noteColumns: ["Notes"] },
            createdAt: null,
            updatedAt: null,
        });
    });

    it("never sends the id of a saved mapping, so an apply cannot rename it", () => {
        const held = adoptSaved(book(), mapping({ path: "URL" }, { sheets: ["Catalog"] }, "m-1"));
        expect(JSON.stringify(requestOf(held))).not.toContain("m-1");
    });

    it("sends a csv without naming a sheet", () => {
        const request = requestOf(openWorkbook([sheet("", 5, ["URL"], mapping({ path: "URL" }))]));
        expect(request?.mapping.options.sheets).toEqual([]);
        expect(request?.sheets).toBeUndefined();
    });

    it("sends nothing while no sheet is chosen", () => {
        expect(requestOf({ ...book(), chosen: [], active: "" })).toBeNull();
    });

    it("names the sheets a request reads and the mappings it saves", () => {
        const one = requestOf(chooseSheet(chooseSheet(book(), "Peptides", false), "Forms", false));
        const several = requestOf(book());
        expect(sheetsOf(one)).toEqual(["Catalog"]);
        expect(sheetsOf(several)).toEqual(["Catalog", "Peptides", "Forms"]);
        expect(sheetsOf(undefined)).toEqual([]);
        expect(savedNames(" Client ", one)).toEqual(["Client"]);
        expect(savedNames("Client", several)).toEqual(["Client / Catalog", "Client / Peptides", "Client / Forms"]);
        expect(savedNames("  ", several)).toEqual([]);
    });

    it("counts the rows of the chosen sheets and of the whole workbook", () => {
        expect(chosenRows(book())).toBe(9);
        expect(chosenRows(chooseSheet(book(), "Forms", false))).toBe(5);
        expect(rowsOf([catalog, peptides, empty, forms])).toBe(9);
        expect(rowsOf([])).toBe(0);
    });
});

describe("what the columns step says", () => {
    it.each([
        {
            name: "every chosen sheet can be read",
            edit: (held: Workbook) => chooseSheet(held, "Forms", false),
            want: { kind: "ready" },
        },
        {
            name: "another chosen sheet still lacks a path or an entity",
            edit: (held: Workbook) => held,
            want: { kind: "otherSheet", sheet: "Forms" },
        },
        {
            name: "the open sheet matched nothing",
            edit: (held: Workbook) => activateSheet(held, "Forms"),
            want: { kind: "unmatched" },
        },
        {
            name: "the open sheet lost the column it matched",
            edit: (held: Workbook) => assignColumn(activateSheet(held, "Peptides"), "Entity", null),
            want: { kind: "needsTarget" },
        },
        {
            name: "the open sheet has no header row and nothing is mapped yet",
            edit: (held: Workbook) => headerless(activateSheet(held, "Forms"), true),
            want: { kind: "needsTarget" },
        },
        {
            name: "no sheet is chosen",
            edit: (held: Workbook) => ({ ...held, chosen: [], active: "" }),
            want: { kind: "noSheet" },
        },
    ])("says so when $name", ({ edit, want }) => {
        expect(columnsNotice(edit(book()))).toEqual(want);
    });

    it("counts group columns as enough to read a sheet", () => {
        const held = setSheetOptions(activateSheet(book(), "Forms"), { levelColumns: ["Name"] });
        expect(columnsNotice(held)).toEqual({ kind: "ready" });
    });
});
