import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Finding, FindingCounts, FindingFilter, PreviewFromPagesRequest, ResolveFindingResponse, ResolveFindingsRequest } from "../../data/types.js";
import { aFinding } from "../../testing/findings.js";
import type { ProposalStart } from "../graph/actions/propose-entities.js";
import type { StartSeed } from "../runs/start-seed.js";

interface Held {
    findings: Finding[];
    counts: FindingCounts;
    filters: FindingFilter[];
    resolved: unknown[];
    answer: Partial<ResolveFindingResponse>;
    ids: string[] | null;
    collected: FindingFilter[];
    resolvedMany: ResolveFindingsRequest[];
}

const fresh = (): Held => ({
    findings: [aFinding()],
    counts: { codes: [{ code: "field_changed_on_site", severity: "warning", count: 1 }], total: 1 },
    filters: [],
    resolved: [],
    answer: {},
    ids: null,
    collected: [],
    resolvedMany: [],
});

const held: Held = fresh();

const idsOf = (): string[] => held.ids ?? held.findings.map((finding) => finding.id);

const pageOfFinding = (findingId: string): string => `page-${findingId}`;

const siteSource = "site" as PreviewFromPagesRequest["source"];

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
    useFindingIds: () => ({ data: idsOf(), isPending: false, error: null }),
    useCollectFindingIds: () => ({
        mutateAsync: (filter: FindingFilter) => {
            held.collected.push(filter);
            return Promise.resolve(idsOf());
        },
    }),
    useResolveFinding: () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answer: Partial<ResolveFindingResponse>) => void }) => {
            held.resolved.push(request);
            options?.onSuccess?.(held.answer);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
    useResolveFindings: () => ({
        mutateAsync: (request: ResolveFindingsRequest) => {
            held.resolvedMany.push(request);
            const proposals: PreviewFromPagesRequest | null =
                request.option === "propose_entity"
                    ? { siteId: "s1", pageIds: (request.findingIds ?? []).map(pageOfFinding), source: siteSource }
                    : null;
            return Promise.resolve({ findings: [], proposals });
        },
    }),
}));

vi.mock("../graph/actions/propose-entities.js", () => ({
    ProposeEntitiesDrawer: ({ open, start }: { open: boolean; start: ProposalStart }) =>
        open && start.kind === "seed" ? (
            <div data-seeded-pages={(start.seed.pageIds ?? []).length} data-seeded-first={start.seed.pageIds?.[0]} data-seeded-source={start.seed.source} />
        ) : null,
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

const withoutEntity = (id: string): Finding =>
    aFinding({
        id,
        pageId: pageOfFinding(id),
        code: "page_without_entity",
        subjectKey: "",
        subject: { path: "/mugs/" },
        options: [
            { name: "propose_entity", kind: "proposal" },
            { name: "ignore", kind: "model" },
        ],
    });

beforeEach(() => {
    Object.assign(held, fresh());
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

    it("opens the entity proposals a proposal option answers, at their preview, instead of settling it", () => {
        held.findings = [withoutEntity("f1")];
        held.answer = { proposals: { siteId: "s1", pageIds: ["page-f1"], source: siteSource } };
        const { container } = inbox();

        expect(container.querySelector("[data-seeded-pages]")).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: said.options.propose_entity }));

        expect(held.resolved).toStrictEqual([{ siteId: "s1", findingId: "f1", option: "propose_entity" }]);
        expect(screen.queryByRole("dialog")).toBeNull();
        const seeded = container.querySelector("[data-seeded-pages]");
        expect(seeded?.getAttribute("data-seeded-pages")).toBe("1");
        expect(seeded?.getAttribute("data-seeded-first")).toBe("page-f1");
        expect(seeded?.getAttribute("data-seeded-source")).toBe("site");
    });

    it("proposes entities for every page of the code the filters show, a batch at a time, with no settling dialog", async () => {
        held.findings = [withoutEntity("f1"), withoutEntity("f2")];
        held.ids = Array.from({ length: 1001 }, (_, at) => `f${at}`);
        held.counts = { codes: [{ code: "page_without_entity", severity: "warning", count: 1001 }], total: 1001 };
        const { container } = inbox("/s/s1/findings?code=page_without_entity");

        expect(screen.queryByRole("button", { name: said.bulk.all(said.options.propose_entity, 1001) })).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: said.bulk.propose(1001) }));

        await waitFor(() => {
            expect(container.querySelector("[data-seeded-pages]")?.getAttribute("data-seeded-pages")).toBe("1001");
        });
        expect(held.collected).toStrictEqual([{ siteId: "s1", status: "open", code: "page_without_entity" }]);
        expect(held.resolvedMany.map((request) => [request.code, request.option, (request.findingIds ?? []).length])).toStrictEqual([
            ["page_without_entity", "propose_entity", 1000],
            ["page_without_entity", "propose_entity", 1],
        ]);
        expect(container.querySelector("[data-seeded-pages]")?.getAttribute("data-seeded-first")).toBe("page-f0");
        expect(screen.queryByRole("dialog")).toBeNull();
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
