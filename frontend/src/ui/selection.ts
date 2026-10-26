export interface PageFilter {
    status?: string;
    entityId?: string;
    includeDescendants?: boolean;
    unmapped?: boolean;
    pathPrefix?: string;
    pageTypeId?: string;
    body?: "with" | "without";
    findings?: boolean;
    search?: string;
    under?: string;
}

export interface SubtreeRequest {
    pageId: string;
    typeId?: string;
}

export interface PageSelection {
    siteId: string;
    pageIds?: string[];
    subtrees?: SubtreeRequest[];
    filter?: PageFilter;
    seen?: number;
    ruleId?: string;
    except?: string[];
}

export interface SelectionRow {
    ancestors: readonly string[];
    typeId?: string | null;
    ruleId?: string | null;
}

export interface SubtreePick {
    pageId: string;
    typeId?: string;
    size?: number;
}

export interface HeldSubtree {
    pageId: string;
    typeId?: string;
    size: number;
    row: SelectionRow;
}

export type Whole = { kind: "filter"; filter: PageFilter; seen: number } | { kind: "rule"; ruleId: string; seen: number };

export interface Selection {
    ids: ReadonlyMap<string, SelectionRow>;
    subtrees: readonly HeldSubtree[];
    whole: Whole | null;
    except: ReadonlyMap<string, SelectionRow>;
    anchor: string | null;
}

export type RowOf = (id: string) => SelectionRow | undefined;

const bare: SelectionRow = Object.freeze({ ancestors: Object.freeze([]) });

export const noSelection: Selection = Object.freeze({
    ids: new Map<string, SelectionRow>(),
    subtrees: Object.freeze([]),
    whole: null,
    except: new Map<string, SelectionRow>(),
    anchor: null,
});

function within(pageId: string, id: string, row: SelectionRow): boolean {
    return id === pageId || row.ancestors.includes(pageId);
}

function ofType(typeId: string | undefined, row: SelectionRow): boolean {
    return typeId === undefined || row.typeId === typeId;
}

function pickHolds(pick: { pageId: string; typeId?: string }, id: string, row: SelectionRow): boolean {
    return within(pick.pageId, id, row) && ofType(pick.typeId, row);
}

function wholeHolds(whole: Whole | null, row: SelectionRow): boolean {
    if (whole === null) {
        return false;
    }
    if (whole.kind === "rule") {
        return row.ruleId === whole.ruleId;
    }
    const wanted = whole.filter.pageTypeId;
    return wanted === undefined || row.typeId === undefined || row.typeId === wanted;
}

function covered(state: Selection, id: string, row: SelectionRow): boolean {
    return state.subtrees.some((pick) => pickHolds(pick, id, row)) || wholeHolds(state.whole, row);
}

function facets(filter: PageFilter): string[] {
    return Object.entries(filter)
        .filter(([, value]) => value !== undefined && value !== "")
        .map(([key]) => key);
}

function pickInside(outer: { pageId: string; typeId?: string }, inner: HeldSubtree): boolean {
    return within(outer.pageId, inner.pageId, inner.row) && (outer.typeId === undefined || outer.typeId === inner.typeId);
}

function wholeContains(whole: Whole | null, pick: HeldSubtree): boolean {
    if (whole === null || whole.kind === "rule") {
        return false;
    }
    const held = facets(whole.filter);
    if (held.length === 0) {
        return true;
    }
    return held.length === 1 && held[0] === "pageTypeId" && whole.filter.pageTypeId === pick.typeId;
}

function wholly(state: Selection, pick: HeldSubtree): boolean {
    return wholeContains(state.whole, pick) || state.subtrees.some((other) => other !== pick && pickInside(other, pick));
}

function settled(state: Selection): Selection {
    const subtrees = state.subtrees.filter((pick) => !wholly(state, pick));
    const kept: Selection = { ...state, subtrees };
    const ids = new Map([...state.ids].filter(([id, row]) => !covered(kept, id, row)));
    const except = new Map([...state.except].filter(([id, row]) => covered(kept, id, row)));
    return { ...kept, ids, except };
}

export function isSelected(state: Selection, id: string, row: SelectionRow = bare): boolean {
    if (state.except.has(id)) {
        return false;
    }
    return state.ids.has(id) || covered(state, id, row);
}

export function isEmpty(state: Selection): boolean {
    return state.ids.size === 0 && state.subtrees.length === 0 && state.whole === null;
}

export function count(state: Selection): number {
    const picked = state.subtrees.reduce((sum, pick) => sum + pick.size, 0);
    const whole = state.whole === null ? 0 : state.whole.seen;
    return Math.max(0, state.ids.size + picked + whole - state.except.size);
}

export function toggle(state: Selection, id: string, row: SelectionRow = bare): Selection {
    const ids = new Map(state.ids);
    const except = new Map(state.except);
    if (isSelected(state, id, row)) {
        if (!ids.delete(id)) {
            except.set(id, row);
        }
    } else if (!except.delete(id)) {
        ids.set(id, row);
    }
    return settled({ ...state, ids, except, anchor: id });
}

function set(state: Selection, id: string, row: SelectionRow, on: boolean): Selection {
    return isSelected(state, id, row) === on ? state : { ...toggle(state, id, row), anchor: state.anchor };
}

export function between<T>(rows: readonly T[], from: T, to: T): readonly T[] {
    const start = rows.indexOf(from);
    const end = rows.indexOf(to);
    if (start < 0 || end < 0) {
        return [];
    }
    return rows.slice(Math.min(start, end), Math.max(start, end) + 1);
}

export function range(state: Selection, rows: readonly string[], target: string, rowOf: RowOf): Selection {
    const anchor = state.anchor;
    const span = anchor === null ? [] : between(rows, anchor, target);
    if (anchor === null || span.length === 0) {
        return toggle(state, target, rowOf(target) ?? bare);
    }
    const on = isSelected(state, anchor, rowOf(anchor) ?? bare);
    let next = state;
    for (const id of span) {
        next = set(next, id, rowOf(id) ?? bare, on);
    }
    return next;
}

function holding(pick: SubtreePick, row: SelectionRow): HeldSubtree {
    const size = pick.size ?? (pick.typeId === undefined ? 1 : 0);
    return pick.typeId === undefined ? { pageId: pick.pageId, size, row } : { pageId: pick.pageId, typeId: pick.typeId, size, row };
}

export function selectSubtree(state: Selection, pick: SubtreePick, row: SelectionRow = bare): Selection {
    const next = holding(pick, row);
    const except = new Map([...state.except].filter(([id, taken]) => !pickHolds(next, id, taken)));
    const cleared: Selection = { ...state, except };
    if (wholly(cleared, next) || cleared.subtrees.some((other) => other.pageId === next.pageId && other.typeId === next.typeId)) {
        return settled(cleared);
    }
    const subtrees = [...cleared.subtrees.filter((other) => !pickInside(next, other)), next];
    return settled({ ...cleared, subtrees });
}

export function clearSubtree(state: Selection, pick: SubtreeRequest): Selection {
    const subtrees = state.subtrees.filter((other) => !pickInside(pick, other));
    const ids = new Map([...state.ids].filter(([id, row]) => !pickHolds(pick, id, row)));
    const except = new Map([...state.except].filter(([id, row]) => !pickHolds(pick, id, row)));
    return settled({ ...state, subtrees, ids, except });
}

export function coveredAbove(state: Selection, id: string, row: SelectionRow): boolean {
    return (
        wholeHolds(state.whole, row) ||
        state.subtrees.some((pick) => pick.pageId !== id && pickHolds(pick, id, row))
    );
}

export function selectAll(state: Selection, filter: PageFilter, seen: number): Selection {
    return settled({ ...state, whole: { kind: "filter", filter, seen }, except: new Map() });
}

export function selectGroup(state: Selection, pageTypeId: string, seen: number): Selection {
    return selectAll(state, { pageTypeId }, seen);
}

export function selectRule(state: Selection, ruleId: string, seen: number): Selection {
    return settled({ ...state, whole: { kind: "rule", ruleId, seen }, except: new Map() });
}

export function clear(): Selection {
    return noSelection;
}

function cleanFilter(filter: PageFilter): PageFilter {
    const out: PageFilter = {};
    for (const [key, value] of Object.entries(filter) as [keyof PageFilter, PageFilter[keyof PageFilter]][]) {
        if (value !== undefined && value !== "") {
            Object.assign(out, { [key]: value });
        }
    }
    return out;
}

export function toRequest(siteId: string, state: Selection): PageSelection {
    const out: PageSelection = { siteId };
    if (state.ids.size > 0) {
        out.pageIds = [...state.ids.keys()];
    }
    if (state.subtrees.length > 0) {
        out.subtrees = state.subtrees.map((pick) => (pick.typeId === undefined ? { pageId: pick.pageId } : { pageId: pick.pageId, typeId: pick.typeId }));
    }
    if (state.whole !== null) {
        if (state.whole.kind === "filter") {
            out.filter = cleanFilter(state.whole.filter);
        } else {
            out.ruleId = state.whole.ruleId;
        }
        out.seen = state.whole.seen;
    }
    if (state.except.size > 0) {
        out.except = [...state.except.keys()];
    }
    return out;
}
