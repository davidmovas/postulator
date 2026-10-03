import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ImportMapping, ImportSheet, InspectResult } from "../../data/types.js";

const inspectMutate = vi.fn();
const previewMutate = vi.fn();
const applyMutate = vi.fn();
const reset = vi.fn();
const held: { inspected: InspectResult | undefined } = { inspected: undefined };

vi.mock("../../data/hooks/imports.js", () => ({
    useInspectSheet: () => ({ mutate: inspectMutate, reset, data: held.inspected, isPending: false, error: null }),
    usePreviewSheet: () => ({ mutate: previewMutate, reset, data: undefined, isPending: false, error: null }),
    useApplySheet: () => ({ mutate: applyMutate, reset, data: undefined, isPending: false, error: null }),
    useSheetSample: () => ({ data: undefined, isLoading: false, error: null }),
}));

const { useImportFlow } = await import("./flow.js");

function mapping(columns: Record<string, string>, sheets: string[], id?: string): ImportMapping {
    return { id, columns, options: { sheets }, createdAt: null, updatedAt: null };
}

function sheet(name: string, rows: number, headers: string[], columns: Record<string, string>): ImportSheet {
    return { name, rows, headers, detected: mapping(columns, name === "" ? [] : [name]) };
}

const catalog = sheet("Catalog", 3, ["URL", "Title"], { path: "URL", title: "Title" });
const compounds = sheet("Compounds", 2, ["Entity", "Keywords"], { entity: "Entity", keywords: "Keywords" });

function inspected(sheets: ImportSheet[]): InspectResult {
    return {
        headers: sheets[0]?.headers ?? [],
        sample: [["/peptides/", "Peptides"]],
        rows: sheets[0]?.rows ?? 0,
        sheets,
        detected: sheets[0]?.detected ?? mapping({}, []),
        saved: null,
    };
}

const path = "C:\\sheets\\client.xlsx";

function lastPreview(): Record<string, unknown> {
    const call = previewMutate.mock.calls[previewMutate.mock.calls.length - 1];
    if (call === undefined) {
        throw new Error("nothing was previewed");
    }
    return call[0] as Record<string, unknown>;
}

function sheetsPreviewed(): string[] | undefined {
    const sent = lastPreview()["sheets"] as { sheet: string }[] | undefined;
    return sent?.map((each) => each.sheet);
}

beforeEach(() => {
    inspectMutate.mockClear();
    previewMutate.mockClear();
    applyMutate.mockClear();
    held.inspected = inspected([catalog, compounds]);
    window.localStorage.clear();
});

describe("the import flow over a workbook", () => {
    it("reads the workbook once and previews every sheet with rows in one request", () => {
        renderHook(() => useImportFlow("s1", path, true));
        expect(inspectMutate).toHaveBeenCalledWith({ siteId: "s1", path, sheets: [], noHeader: false });
        expect(sheetsPreviewed()).toEqual(["Catalog", "Compounds"]);
        expect(lastPreview()["mapping"]).toEqual({ columns: null, options: {}, createdAt: null, updatedAt: null });
    });

    it("previews one sheet left on as the one mapping of the file, and brings the other back with its settings", () => {
        const { result } = renderHook(() => useImportFlow("s1", path, true));
        act(() => {
            result.current.activate("Compounds");
        });
        act(() => {
            result.current.assign("Keywords", "anchors");
        });
        act(() => {
            result.current.choose("Compounds", false);
        });
        expect(sheetsPreviewed()).toBeUndefined();
        expect(lastPreview()["mapping"]).toEqual({
            columns: { path: "URL", title: "Title" },
            options: { sheets: ["Catalog"] },
            createdAt: null,
            updatedAt: null,
        });

        act(() => {
            result.current.choose("Compounds", true);
        });
        const sent = lastPreview()["sheets"] as { sheet: string; mapping: ImportMapping }[];
        expect(sent.map((each) => each.sheet)).toEqual(["Catalog", "Compounds"]);
        expect(sent[1]?.mapping.columns).toEqual({ entity: "Entity", anchors: "Keywords" });
    });

    it("does not preview again until the request changes", () => {
        const { result, rerender } = renderHook(() => useImportFlow("s1", path, true));
        rerender();
        act(() => {
            result.current.activate("Compounds");
        });
        expect(previewMutate).toHaveBeenCalledOnce();
    });

    it("previews nothing while it is not on the preview step", () => {
        renderHook(() => useImportFlow("s1", path, false));
        expect(previewMutate).not.toHaveBeenCalled();
    });

    it("applies the same request it previewed, saving a mapping per sheet when asked", () => {
        const { result } = renderHook(() => useImportFlow("s1", path, true));
        act(() => {
            result.current.setSaveAs(" Client ");
        });
        act(() => {
            result.current.startApply();
        });
        const sent = applyMutate.mock.calls[0]?.[0] as Record<string, unknown>;
        expect(sent["options"]).toEqual({ saveMappingAs: "Client" });
        expect((sent["sheets"] as { sheet: string }[]).map((each) => each.sheet)).toEqual(["Catalog", "Compounds"]);
        expect(sent["mapping"]).toEqual(lastPreview()["mapping"]);
    });

    it("applies without saving a mapping when no name was given", () => {
        const { result } = renderHook(() => useImportFlow("s1", path, false));
        act(() => {
            result.current.startApply();
        });
        expect((applyMutate.mock.calls[0]?.[0] as Record<string, unknown>)["options"]).toEqual({});
    });

    it("puts a saved mapping picked before the file was read on the sheet it names once the workbook is read", () => {
        held.inspected = undefined;
        const { result, rerender } = renderHook(() => useImportFlow("s1", path, false));
        act(() => {
            result.current.adopt(mapping({ entity: "Name" }, ["Compounds"], "m-1"));
        });
        held.inspected = inspected([catalog, compounds]);
        rerender();
        expect(result.current.book?.active).toBe("Compounds");
        expect(result.current.book?.perSheet["Compounds"]?.columns).toEqual({ entity: "Name" });
        expect(result.current.book?.perSheet["Catalog"]?.columns).toEqual({ path: "URL", title: "Title" });
    });

    it("reads the first sheet's sample from the workbook and another sheet's columns from its own header row", () => {
        const { result } = renderHook(() => useImportFlow("s1", path, false));
        expect(result.current.reading.headers).toEqual(["URL", "Title"]);
        expect(result.current.reading.sample).toEqual([["/peptides/", "Peptides"]]);
        expect(result.current.detected).toEqual({ path: "URL", title: "Title" });
        act(() => {
            result.current.activate("Compounds");
        });
        expect(result.current.reading.headers).toEqual(["Entity", "Keywords"]);
        expect(result.current.reading.sample).toEqual([]);
    });

    it("forgets what a sheet without a header row matched", () => {
        const { result } = renderHook(() => useImportFlow("s1", path, false));
        act(() => {
            result.current.setHeaderless(true);
        });
        expect(result.current.detected).toBeNull();
        expect(result.current.book?.perSheet["Catalog"]?.columns).toEqual({});
    });

    it("reads the file again when the same file is chosen after another import", () => {
        const { rerender } = renderHook(({ at }) => useImportFlow("s1", at, false), { initialProps: { at: path } });
        rerender({ at: "" });
        rerender({ at: path });
        expect(inspectMutate).toHaveBeenCalledTimes(2);
    });

    it("remembers the file with the rows of every sheet", () => {
        renderHook(() => useImportFlow("s1", path, false));
        const { result } = renderHook(() => useImportFlow("s1", "", false));
        expect(result.current.recent[0]).toMatchObject({ path, rows: 5 });
    });
});
