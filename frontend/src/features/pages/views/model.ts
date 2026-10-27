import { copy } from "../../../copy/index.js";
import type { Page, PageFacets, SiteMapNode } from "../../../data/types.js";
import type { ColumnLayout, Tracks } from "../../../ui/collapse.js";
import { laidOut, widestFitting } from "../../../ui/collapse.js";
import type { ChildrenOf, Selection, SelectionRow, TickState } from "../../../ui/selection.js";
import { isEmpty, isSelected } from "../../../ui/selection.js";
import type { Branches } from "../../map/model/tree.js";
import { reasonLabel } from "../../settings/types/labels.js";
import { pageStatusLabel } from "../labels.js";

const said = copy.pages;

export interface RuleRef {
    number: number;
    name: string;
}

export function typeReasonLine(page: Pick<Page, "typeSource" | "typeReason" | "typeRuleId">, ruleOf: (ruleId: string) => RuleRef | undefined): string {
    switch (page.typeReason) {
        case "rule": {
            const rule = ruleOf(page.typeRuleId);
            return rule === undefined ? said.typeReason.ruleGone : said.typeReason.rule(rule.number, rule.name === "" ? copy.classification.unnamed : rule.name);
        }
        case "unmatched":
            return said.typeReason.unmatched;
        case "user":
            return said.typeReason.user;
        case "ai":
            return said.typeReason.ai;
        case "import":
            return said.typeReason.import;
        case "":
            return said.typeReason.none;
        default:
            return reasonLabel(page.typeReason);
    }
}

export function pageRow(page: Pick<Page, "pageTypeId" | "typeRuleId">): SelectionRow {
    return { typeId: page.pageTypeId, ruleId: page.typeRuleId === "" ? null : page.typeRuleId };
}

export type TypeIdOf = (typeName: string) => string | null;

export function nodeRow(node: SiteMapNode, ancestors: readonly string[], typeIdOf: TypeIdOf): SelectionRow {
    return { ancestors, typeId: node.page.type === "" ? null : typeIdOf(node.page.type) };
}

export function nodeSize(node: SiteMapNode): number {
    return node.descendants + 1;
}

export function childrenFrom(branches: Branches, chainOf: (pageId: string) => readonly string[] | undefined, typeIdOf: TypeIdOf, within: string | undefined): ChildrenOf {
    return (pageId) => {
        const chain = chainOf(pageId);
        const branch = branches.get(pageId);
        if (chain === undefined || branch === undefined || branch.pending || branch.hasMore) {
            return null;
        }
        const ancestors = [pageId, ...chain];
        return branch.nodes.map((node) => ({
            id: node.page.pageId,
            row: nodeRow(node, ancestors, typeIdOf),
            size: within === undefined ? nodeSize(node) : null,
        }));
    };
}

export interface Naming {
    typeName: (typeId: string) => string;
    pagePath: (pageId: string) => string;
    ruleName: (ruleId: string) => string;
    entityName: (entityId: string) => string;
}

export function filterWords(filter: PageFacets, naming: Naming): string {
    const parts: string[] = [];
    if ((filter.pageTypeId ?? "") !== "") {
        parts.push(said.describe.type(naming.typeName(filter.pageTypeId ?? "")));
    }
    if ((filter.status ?? "") !== "") {
        parts.push(said.describe.status(pageStatusLabel(filter.status ?? "")));
    }
    if (filter.body === "with") {
        parts.push(said.describe.withText);
    }
    if (filter.body === "without") {
        parts.push(said.describe.withoutText);
    }
    if (filter.findings === true) {
        parts.push(said.describe.findings);
    }
    if ((filter.under ?? "") !== "") {
        parts.push(said.describe.under(naming.pagePath(filter.under ?? "")));
    }
    if ((filter.pathPrefix ?? "") !== "") {
        parts.push(said.describe.prefix(filter.pathPrefix ?? ""));
    }
    if ((filter.search ?? "") !== "") {
        parts.push(said.describe.search(filter.search ?? ""));
    }
    if ((filter.entityId ?? "") !== "") {
        parts.push(said.describe.entity(naming.entityName(filter.entityId ?? ""), filter.includeDescendants === true));
    }
    if (filter.unmapped === true) {
        parts.push(said.describe.unmapped);
    }
    return parts.length === 0 ? said.describe.everyPage : said.describe.every(parts);
}

export function selectionWords(state: Selection, naming: Naming): string {
    const parts: string[] = [];
    if (state.whole !== null) {
        if (state.whole.kind === "rule") {
            parts.push(said.describe.rule(naming.ruleName(state.whole.ruleId)));
        } else {
            const facets = Object.keys(state.whole.filter).filter((key) => {
                const value = (state.whole?.kind === "filter" ? state.whole.filter : {})[key as keyof PageFacets];
                return value !== undefined && value !== null && value !== "";
            });
            parts.push(
                facets.length === 1 && facets[0] === "pageTypeId"
                    ? said.describe.group(naming.typeName(state.whole.filter.pageTypeId ?? ""))
                    : filterWords(state.whole.filter, naming),
            );
        }
    }
    for (const pick of state.subtrees) {
        parts.push(
            pick.typeId === undefined
                ? said.describe.subtree(naming.pagePath(pick.pageId))
                : said.describe.subtreeOf(naming.typeName(pick.typeId), naming.pagePath(pick.pageId)),
        );
    }
    if (state.ids.size > 0) {
        parts.push(said.describe.picked(state.ids.size));
    }
    const removed: string[] = [];
    if (state.except.size > 0) {
        removed.push(said.describe.except(state.except.size));
    }
    if (state.outs.length > 0) {
        removed.push(said.describe.outs(state.outs.map((out) => naming.pagePath(out.pageId))));
    }
    return said.describe.selection(parts, removed);
}

export function prefillOf(count: number | null, words: string, paths: readonly string[]): string {
    return said.bulk.prefill(count, words, paths);
}

export type ListColumn = "box" | "path" | "title" | "type" | "status" | "entity" | "created" | "drift" | "synced";

export type ListLayout = ColumnLayout<ListColumn>;

const tracks: Tracks<ListColumn> = {
    box: { track: "20px", least: 20 },
    path: { track: "minmax(140px,2.4fr)", least: 140 },
    title: { track: "minmax(120px,2fr)", least: 120 },
    type: { track: "minmax(84px,1fr)", least: 84 },
    status: { track: "84px", least: 84 },
    entity: { track: "minmax(96px,1.4fr)", least: 96 },
    created: { track: "76px", least: 76 },
    drift: { track: "32px", least: 32 },
    synced: { track: "72px", least: 72 },
};

const shrinking: Tracks<ListColumn> = {
    ...tracks,
    path: { track: "minmax(0,2fr)", least: 0 },
    type: { track: "minmax(0,1fr)", least: 0 },
};

const rowChrome = 24;

const layouts: readonly ListLayout[] = [
    laidOut(["box", "path", "title", "type", "status", "entity", "created", "drift", "synced"], tracks, rowChrome),
    laidOut(["box", "path", "title", "type", "status", "entity", "created", "drift"], tracks, rowChrome),
    laidOut(["box", "path", "title", "type", "status", "drift"], tracks, rowChrome),
    laidOut(["box", "path", "type", "status"], tracks, rowChrome),
];

const narrowest = laidOut<ListColumn>(["box", "path", "type"], shrinking, rowChrome);

export function listLayout(width: number): ListLayout {
    return widestFitting(width, layouts, narrowest);
}

function sameFacets(left: PageFacets, right: PageFacets): boolean {
    const keys = new Set([...Object.keys(left), ...Object.keys(right)]) as Set<keyof PageFacets>;
    return [...keys].every((key) => (left[key] ?? "") === (right[key] ?? ""));
}

export function wholeOf(state: Selection, facets: PageFacets): boolean {
    return state.whole !== null && state.whole.kind === "filter" && sameFacets(state.whole.filter, facets);
}

export function headState(state: Selection, facets: PageFacets): TickState {
    if (wholeOf(state, facets)) {
        return state.except.size === 0 && state.outs.length === 0 ? "on" : "some";
    }
    return isEmpty(state) ? "off" : "some";
}

export function groupState(state: Selection, typeId: string): TickState {
    if (wholeOf(state, { pageTypeId: typeId })) {
        return state.except.size === 0 && state.outs.length === 0 ? "on" : "some";
    }
    const held =
        [...state.ids.values()].some((row) => row.typeId === typeId) || state.subtrees.some((pick) => pick.typeId === typeId);
    return held ? "some" : "off";
}

export function withoutType(state: Selection, typeId: string): Selection {
    const whole = wholeOf(state, { pageTypeId: typeId }) ? null : state.whole;
    return {
        ...state,
        whole,
        ids: new Map([...state.ids].filter(([, row]) => row.typeId !== typeId)),
        except: whole === null ? new Map() : new Map([...state.except].filter(([, row]) => row.typeId !== typeId)),
        subtrees: state.subtrees.filter((pick) => pick.typeId !== typeId),
        outs: state.outs.filter((out) => out.typeId !== typeId),
    };
}

export interface KnownRow {
    path: string;
    row: SelectionRow;
}

export function selectedPaths(state: Selection, known: ReadonlyMap<string, KnownRow>, limit: number): string[] {
    const out: string[] = [];
    for (const [id, held] of known) {
        if (out.length >= limit) {
            break;
        }
        if (isSelected(state, id, held.row)) {
            out.push(held.path);
        }
    }
    return out;
}
