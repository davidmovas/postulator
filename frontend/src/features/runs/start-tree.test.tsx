import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { TransportError } from "../../data/errors.js";
import type { AddedPage, SiteMapNode } from "../../data/types.js";
import { answerSiteMap, resetSiteMap } from "../../testing/site-map.js";
import { aNode, aRef } from "../map/model/fixture.js";

interface Held {
    asked: string[][];
    added: AddedPage[];
    error: unknown;
}

const held: Held = { asked: [], added: [], error: null };

vi.mock("../../data/hooks/pages.js", async () => {
    const fake = await import("../../testing/site-map.js");
    return { useSiteMap: fake.fakeSiteMap };
});

vi.mock("../../data/hooks/runs.js", () => ({
    useRunParents: (siteId: string, pageIds: readonly string[]) => {
        if (pageIds.length > 0) {
            held.asked.push([siteId, ...pageIds]);
        }
        return { data: pageIds.length === 0 ? undefined : { added: held.added }, error: pageIds.length === 0 ? null : held.error };
    },
}));

const { TargetTree } = await import("./start-tree.js");

const said = copy.runs.start;

const missing = (pageId: string, path: string, overrides: Partial<SiteMapNode> = {}): SiteMapNode =>
    aNode(pageId, path, { status: "planned" as SiteMapNode["status"], page: aRef(pageId, path, { onSite: false }), ...overrides });

const bikes = aNode("bikes", "/bikes/", { entityId: "e1", children: 1, descendants: 3 });
const cargo = missing("cargo", "/bikes/cargo/", { entityId: "e2", children: 2, descendants: 2 });
const max = missing("max", "/bikes/cargo/max/", { entityId: "e3" });
const mini = missing("mini", "/bikes/cargo/mini/");

function Harness({ problem = null }: { problem?: string | null }) {
    const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
    return <TargetTree siteId="s1" selected={selected} problem={problem} onChange={(next) => setSelected(next)} />;
}

function rowOf(pageId: string): HTMLElement {
    const row = document.querySelector<HTMLElement>(`[data-page-pick="${pageId}"]`);
    if (row === null) {
        throw new Error(`no row for ${pageId}`);
    }
    return row;
}

function openToMax(): void {
    fireEvent.click(within(rowOf("bikes")).getByRole("button", { name: copy.pages.pick.expand }));
    fireEvent.click(within(rowOf("cargo")).getByRole("button", { name: copy.pages.pick.expand }));
}

function refused(cause: TransportError): Error {
    return Object.assign(new Error(cause.message), { cause });
}

beforeEach(() => {
    resetSiteMap();
    held.asked = [];
    held.added = [];
    held.error = null;
    answerSiteMap({}, [bikes]);
    answerSiteMap({ parentId: "bikes" }, [cargo]);
    answerSiteMap({ parentId: "cargo" }, [max, mini]);
});

describe("the run start's targets", () => {
    it("asks the run which parents it writes first for the settled pick, and says so on each", async () => {
        held.added = [{ pageId: "cargo", path: "/bikes/cargo/", neededBy: "/bikes/cargo/max/" }];
        render(<Harness />);
        openToMax();

        fireEvent.click(screen.getByRole("checkbox", { name: "max/" }));

        await waitFor(() => {
            expect(held.asked).toStrictEqual([["s1", "max"]]);
        });
        await waitFor(() => {
            expect(within(rowOf("cargo")).getByText(said.writtenFirst)).toBeTruthy();
        });
        const parent = screen.getByRole("checkbox", { name: "cargo/" }) as HTMLInputElement;
        expect(parent.checked).toBe(true);
        expect(parent.disabled).toBe(true);
        expect(rowOf("cargo").getAttribute("title")).toBe(said.required("/bikes/cargo/max/"));
        expect(screen.getByText(said.picked(1, 1))).toBeTruthy();
    });

    it("refuses a page nothing is mapped to", () => {
        render(<Harness />);
        openToMax();

        expect((screen.getByRole("checkbox", { name: "mini/" }) as HTMLInputElement).disabled).toBe(true);
        expect(rowOf("mini").getAttribute("title")).toBe(said.unmapped);
    });

    it("says in words why the run cannot write a picked page's parent first", async () => {
        const sentence = "the parent /bikes/cargo/ of /bikes/cargo/max/ is not on the site and nothing is mapped to it";
        held.error = refused({
            code: "INVALID",
            message: sentence,
            details: { field: "pageIds", parentPath: "/bikes/cargo/", path: "/bikes/cargo/max/" },
        });
        render(<Harness />);
        openToMax();

        fireEvent.click(screen.getByRole("checkbox", { name: "max/" }));

        await waitFor(() => {
            expect(screen.getByRole("alert").textContent).toBe(sentence);
        });
    });

    it("shows the start's own refusal before the parents'", () => {
        render(<Harness problem="Select at least one page." />);

        expect(screen.getByRole("alert").textContent).toBe("Select at least one page.");
    });
});
