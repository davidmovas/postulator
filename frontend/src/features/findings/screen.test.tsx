import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Finding, FindingCounts, FindingFilter, ResolveFindingResponse } from "../../data/types.js";
import { aFinding } from "../../testing/findings.js";
import type { StartSeed } from "../runs/start-seed.js";

interface Held {
    findings: Finding[];
    counts: FindingCounts;
    filters: FindingFilter[];
    resolved: unknown[];
    answer: Partial<ResolveFindingResponse>;
}

const held: Held = { findings: [], counts: { codes: [], total: 0 }, filters: [], resolved: [], answer: {} };

const sentence = "Take what the site holds for the SEO description of /mugs/; nothing is written to the site";

vi.mock("../../data/hooks/findings.js", () => ({
    useFindings: (filter: FindingFilter) => {
        held.filters.push(filter);
        return {
            data: { pages: [{ items: held.findings, hasMore: false }] },
            isPending: false,
            error: null,
            hasNextPage: false,
            fetchNextPage: () => undefined,
            refetch: () => undefined,
        };
    },
    useFindingCounts: () => ({ data: held.counts, refetch: () => undefined }),
    useExplanation: (request: unknown) => ({
        data: request === null ? undefined : { sentence, kind: "model" },
        isPending: false,
        error: null,
    }),
    useFindingIds: () => ({ data: held.findings.map((finding) => finding.id), isPending: false, error: null }),
    useResolveFinding: () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answer: Partial<ResolveFindingResponse>) => void }) => {
            held.resolved.push(request);
            options?.onSuccess?.(held.answer);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
    useResolveFindings: () => ({ mutateAsync: () => Promise.resolve({ findings: [] }) }),
}));

vi.mock("../../data/hooks/pages.js", () => ({
    usePage: (id: string | null) => ({ data: id === null ? undefined : { page: { id, path: "/mugs/" } }, isPending: false }),
    useSiteMap: () => ({ data: undefined, isPending: false, hasNextPage: false }),
}));

vi.mock("../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types: [] }, isPending: false }),
}));

vi.mock("../runs/start.js", () => ({
    StartRunDrawer: ({ open, seed }: { open: boolean; seed: StartSeed | null }) =>
        open ? <div data-seeded-kind={seed?.kind} data-seeded-fields={seed?.fields.join(",")} /> : null,
}));

vi.mock("../sites/sync-dialog.js", () => ({
    SyncDialog: ({ request }: { request: unknown }) => (request === null ? null : <div data-sync-asked={true} />),
}));

const { FindingsScreen } = await import("./screen.js");

const said = copy.findings;

function inbox(at = "/s/s1/findings") {
    return render(
        <MemoryRouter initialEntries={[at]}>
            <Routes>
                <Route path="/s/:siteId/findings" element={<FindingsScreen />} />
            </Routes>
        </MemoryRouter>,
    );
}

const differs = (id: string): Finding =>
    aFinding({
        id,
        code: "parent_differs",
        subject: { sheet: ["Peptides"], site: [] },
        options: [
            { name: "accept_site", kind: "model" },
            { name: "accept_workbook", kind: "model" },
            { name: "ignore", kind: "model" },
        ],
    });

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

beforeEach(() => {
    held.findings = [aFinding()];
    held.counts = { codes: [{ code: "field_changed_on_site", severity: "warning", count: 1 }], total: 1 };
    held.filters = [];
    held.resolved = [];
    held.answer = {};
});

describe("the findings inbox", () => {
    it("words each open finding: its code, its page, its subject and one button per option", () => {
        inbox();

        const row = within(screen.getByRole("article", { name: "Field edited on the site" }));
        expect(row.getByText("Mugs")).toBeDefined();
        expect(row.getByText("/mugs/")).toBeDefined();
        expect(row.getByText("The SEO description was edited on the site since it was last read or written.")).toBeDefined();
        const take = row.getByRole("button", { name: said.options.accept_site });
        expect(take.getAttribute("title")).toBe(said.optionDoes.accept_site);
        expect(row.getByRole("button", { name: said.options.rewrite_field }).getAttribute("title")).toBe(said.optionDoes.rewrite_field);
        expect(row.getByRole("button", { name: said.options.ignore })).toBeDefined();
    });

    it("lists the codes with their counts and narrows the list to one", () => {
        inbox();

        const rail = within(screen.getByRole("navigation", { name: said.codes }));
        expect(rail.getByRole("button", { name: /^All\s*1$/ })).toBeDefined();
        fireEvent.click(rail.getByRole("button", { name: /^Field edited on the site\s*1$/ }));

        expect(held.filters.at(-1)).toStrictEqual({ siteId: "s1", status: "open", code: "field_changed_on_site" });
    });

    it("reads its filters from the address, so the site map can link to a subtree", () => {
        inbox("/s/s1/findings?status=dismissed&under=p9");

        expect(held.filters.at(-1)).toStrictEqual({ siteId: "s1", status: "dismissed", under: "p9" });
        expect(screen.getByRole("button", { name: said.underRemove("/mugs/") })).toBeDefined();
    });

    it("confirms a model option in a dialog that shows the agent's card body and the sentence of what changes", () => {
        inbox();

        fireEvent.click(screen.getByRole("button", { name: said.options.accept_site }));

        const dialog = within(screen.getByRole("dialog", { name: said.options.accept_site }));
        expect(dialog.getByText(copy.agent.describe.findings.resolve("Field edited on the site").trim())).toBeDefined();
        expect(dialog.getByText("/mugs/")).toBeDefined();
        expect(dialog.getByText(sentence)).toBeDefined();
        fireEvent.click(dialog.getByRole("button", { name: said.resolve.confirm }));

        expect(held.resolved).toStrictEqual([{ siteId: "s1", findingId: "f1", option: "accept_site" }]);
    });

    it("opens the run start a run option answers instead of settling it", () => {
        held.answer = { start: { siteId: "s1", pageIds: ["p1"], kind: "", budget: { maxUsd: 0, maxTokens: 0 }, fields: ["seo_description"], rewrite: ["p1"] } };
        const { container } = inbox();

        fireEvent.click(screen.getByRole("button", { name: said.options.rewrite_field }));

        expect(held.resolved).toStrictEqual([{ siteId: "s1", findingId: "f1", option: "rewrite_field" }]);
        expect(screen.queryByRole("dialog")).toBeNull();
        const seeded = container.querySelector("[data-seeded-kind]");
        expect(seeded?.getAttribute("data-seeded-kind")).toBe("generate");
        expect(seeded?.getAttribute("data-seeded-fields")).toBe("seo_description");
    });

    it("offers every bulk option of one open code under the filters, counted", () => {
        held.findings = [differs("f1"), differs("f2"), differs("f3")];
        held.counts = { codes: [{ code: "parent_differs", severity: "warning", count: 3 }], total: 3 };
        inbox("/s/s1/findings?code=parent_differs");

        for (const option of ["ignore", "accept_site", "accept_workbook"] as const) {
            expect(screen.getByRole("button", { name: said.bulk.all(said.options[option], 3) })).toBeDefined();
        }
        fireEvent.click(screen.getByRole("button", { name: said.bulk.all(said.options.ignore, 3) }));
        const dialog = within(screen.getByRole("dialog", { name: said.bulk.title(said.options.ignore) }));
        expect(dialog.getByText(sentence)).toBeDefined();
        expect(dialog.getByRole("button", { name: said.bulk.confirm(3) })).toBeDefined();
    });

    it("offers no bulk action without one code, or outside the open findings", () => {
        held.findings = [differs("f1")];
        held.counts = { codes: [{ code: "parent_differs", severity: "warning", count: 1 }], total: 1 };
        inbox();
        expect(screen.queryByRole("button", { name: said.bulk.all(said.options.ignore, 1) })).toBeNull();
    });

    it("says who settled a finding, when and how", () => {
        held.findings = [aFinding({ status: "dismissed", resolution: "ignore", resolvedBy: "agent" })];
        inbox("/s/s1/findings?status=dismissed");

        const row = within(screen.getByRole("article", { name: "Field edited on the site" }));
        expect(row.getByText(/^Settled by the agent .*: ignored$/)).toBeDefined();
        expect(row.queryByRole("button", { name: said.options.ignore })).toBeNull();
    });

    it("explains an empty inbox and offers to widen a narrowed one", () => {
        held.findings = [];
        inbox("/s/s1/findings?code=parent_cycle");

        expect(screen.getByText(said.empty.openTitle)).toBeDefined();
        fireEvent.click(screen.getByRole("button", { name: said.empty.filtered }));
        expect(held.filters.at(-1)).toStrictEqual({ siteId: "s1", status: "open" });
    });
});
