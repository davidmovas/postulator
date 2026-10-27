import type { PageFacets, PageSelection, PageSubtree } from "../data/types.js";

export interface SelectionRow {
    ancestors?: readonly string[];
    typeId?: string | null;
    ruleId?: string | null;
}

export interface SubtreePick {
    pageId: string;
    typeId?: string;
    size?: number | null;
}

export interface HeldSubtree {
    pageId: string;
    typeId?: string;
    size: number | null;
    row: SelectionRow;
}

export interface ChildRow {
    id: string;
    row: SelectionRow;
    size: number | null;
}

export type ChildrenOf = (pageId: string) => readonly ChildRow[] | null;

export type RowOf = (id: string) => SelectionRow | undefined;

export type Whole = { kind: "filter"; filter: PageFacets; seen: number } | { kind: "rule"; ruleId: string; seen: number };

export interface Selection {
    ids: ReadonlyMap<string, SelectionRow>;
    subtrees: readonly HeldSubtree[];
    whole: Whole | null;
    except: ReadonlyMap<string, SelectionRow>;
    outs: readonly HeldSubtree[];
    anchor: string | null;
}

export type TickState = "on" | "off" | "some";

export interface TickScope {
    typeId?: string;
    leaf: boolean;
}

interface Scope {
    pageId: string;
    typeId?: string;
}

const bare: SelectionRow = Object.freeze({});

const unknownChildren: ChildrenOf = () => null;

export const noSelection: Selection = Object.freeze({
    ids: new Map<string, SelectionRow>(),
    subtrees: Object.freeze([]),
    whole: null,
    except: new Map<string, SelectionRow>(),
    outs: Object.freeze([]),
    anchor: null,
});

function chainOf(row: SelectionRow): readonly string[] {
    return row.ancestors ?? [];
}

function within(pageId: string, id: string, row: SelectionRow): boolean {
    return id === pageId || chainOf(row).includes(pageId);
}

function below(pageId: string, id: string, row: SelectionRow): boolean {
    return id !== pageId && chainOf(row).includes(pageId);
}

function ofType(typeId: string | undefined, row: SelectionRow): boolean {
    return typeId === undefined || row.typeId === typeId;
}

function holds(scope: Scope, id: string, row: SelectionRow): boolean {
    return within(scope.pageId, id, row) && ofType(scope.typeId, row);
}

function contains(outer: Scope, inner: HeldSubtree): boolean {
    return within(outer.pageId, inner.pageId, inner.row) && (outer.typeId === undefined || outer.typeId === inner.typeId);
}

function reaches(outer: Scope, inner: HeldSubtree): boolean {
    return (
        within(outer.pageId, inner.pageId, inner.row) &&
        (outer.typeId === undefined || inner.typeId === undefined || outer.typeId === inner.typeId)
    );
}

function same(left: Scope, right: Scope): boolean {
    return left.pageId === right.pageId && left.typeId === right.typeId;
}

function facets(filter: PageFacets): string[] {
    return Object.entries(filter)
        .filter(([, value]) => value !== undefined && value !== null && value !== "")
        .map(([key]) => key);
}

function wholeHolds(whole: Whole | null, row: SelectionRow): boolean {
    if (whole === null) {
        return false;
    }
    if (whole.kind === "rule") {
        return row.ruleId === whole.ruleId;
    }
    const wanted = whole.filter.pageTypeId;
    return wanted === undefined || wanted === "" || row.typeId === undefined || row.typeId === wanted;
}

function wholeCovers(whole: Whole | null, typeId: string | undefined): boolean {
    if (whole === null || whole.kind === "rule") {
        return false;
    }
    const held = facets(whole.filter);
    return held.length === 0 || (held.length === 1 && held[0] === "pageTypeId" && whole.filter.pageTypeId === typeId);
}

function covered(state: Selection, id: string, row: SelectionRow): boolean {
    return state.subtrees.some((pick) => holds(pick, id, row)) || wholeHolds(state.whole, row);
}

function outOf(state: Selection, id: string, row: SelectionRow): HeldSubtree | undefined {
    return state.outs.find((out) => holds(out, id, row));
}

function settled(state: Selection): Selection {
    const subtrees = state.subtrees.filter(
        (pick) =>
            !wholeCovers(state.whole, pick.typeId) &&
            !state.subtrees.some((other) => other !== pick && contains(other, pick)) &&
            !state.outs.some((out) => contains(out, pick)),
    );
    const outs = state.outs.filter(
        (out) =>
            !state.outs.some((other) => other !== out && contains(other, out)) &&
            (state.whole !== null || subtrees.some((pick) => reaches(pick, out))),
    );
    const kept: Selection = { ...state, subtrees, outs };
    const ids = new Map([...state.ids].filter(([id, row]) => !covered(kept, id, row) && outOf(kept, id, row) === undefined));
    const except = new Map([...state.except].filter(([id, row]) => covered(kept, id, row) && outOf(kept, id, row) === undefined));
    return { ...kept, ids, except };
}

export function isSelected(state: Selection, id: string, row: SelectionRow = bare): boolean {
    if (state.except.has(id) || outOf(state, id, row) !== undefined) {
        return false;
    }
    return state.ids.has(id) || covered(state, id, row);
}

export function isEmpty(state: Selection): boolean {
    return state.ids.size === 0 && state.subtrees.length === 0 && state.whole === null;
}

export function count(state: Selection): number {
    const picked = state.subtrees.reduce((sum, pick) => sum + (pick.size ?? 0), 0);
    const cut = state.outs.reduce((sum, out) => sum + (out.size ?? 0), 0);
    const whole = state.whole === null ? 0 : state.whole.seen;
    return Math.max(0, state.ids.size + picked + whole - state.except.size - cut);
}

export function exact(state: Selection): boolean {
    if (state.subtrees.some((pick) => pick.size === null) || state.outs.some((out) => out.size === null)) {
        return false;
    }
    const placed = state.subtrees.length > 0 || state.outs.length > 0;
    if (!placed) {
        return true;
    }
    if (state.whole !== null) {
        return false;
    }
    if ([...state.ids.values(), ...state.except.values()].some((row) => row.ancestors === undefined)) {
        return false;
    }
    if (state.subtrees.some((pick) => state.subtrees.some((other) => other !== pick && reaches(other, pick)))) {
        return false;
    }
    return state.outs.every((out) =>
        state.subtrees.some((pick) => pick.typeId === out.typeId && within(pick.pageId, out.pageId, out.row)),
    );
}

function rowAmong(children: readonly ChildRow[] | null, id: string): SelectionRow | undefined {
    return children?.find((child) => child.id === id)?.row;
}

function opened(state: Selection, out: HeldSubtree, id: string, row: SelectionRow, childrenOf: ChildrenOf): Selection | null {
    const outs = state.outs.filter((held) => held !== out);
    if (out.pageId === id) {
        return { ...state, outs };
    }
    const chain = chainOf(row);
    const at = chain.indexOf(out.pageId);
    if (at < 0) {
        return null;
    }
    const path = chain.slice(0, at + 1).reverse();
    const except = new Map(state.except);
    let above: readonly ChildRow[] | null = null;
    for (const [step, node] of path.entries()) {
        const next = path[step + 1] ?? id;
        const children = childrenOf(node);
        if (children === null) {
            return null;
        }
        const nodeRow = step === 0 ? out.row : rowAmong(above, node);
        if (nodeRow === undefined) {
            return null;
        }
        if (ofType(out.typeId, nodeRow)) {
            except.set(node, nodeRow);
        }
        for (const child of children) {
            if (child.id !== next) {
                outs.push(cutAt(out.typeId, child));
            }
        }
        above = children;
    }
    return { ...state, outs, except };
}

function cutAt(typeId: string | undefined, child: ChildRow): HeldSubtree {
    const size = typeId === undefined ? child.size : null;
    return typeId === undefined ? { pageId: child.id, size, row: child.row } : { pageId: child.id, typeId, size, row: child.row };
}

export function toggle(state: Selection, id: string, row: SelectionRow = bare, childrenOf: ChildrenOf = unknownChildren): Selection | null {
    if (isSelected(state, id, row)) {
        const ids = new Map(state.ids);
        const except = new Map(state.except);
        if (!ids.delete(id)) {
            except.set(id, row);
        }
        return settled({ ...state, ids, except, anchor: id });
    }
    const out = outOf(state, id, row);
    if (out !== undefined) {
        const open = opened(state, out, id, row, childrenOf);
        const children = childrenOf(id);
        if (open === null || children === null) {
            return null;
        }
        const shut: Selection = { ...open, outs: [...open.outs, ...children.map((child) => cutAt(out.typeId, child))] };
        const ids = new Map(shut.ids);
        if (!isSelected(shut, id, row)) {
            ids.set(id, row);
        }
        return settled({ ...shut, ids, anchor: id });
    }
    const ids = new Map(state.ids);
    const except = new Map(state.except);
    if (!except.delete(id)) {
        ids.set(id, row);
    }
    return settled({ ...state, ids, except, anchor: id });
}

function set(state: Selection, id: string, row: SelectionRow, on: boolean, childrenOf: ChildrenOf): Selection {
    if (isSelected(state, id, row) === on) {
        return state;
    }
    const next = toggle(state, id, row, childrenOf);
    return next === null ? state : { ...next, anchor: state.anchor };
}

export function between<T>(rows: readonly T[], from: T, to: T): readonly T[] {
    const start = rows.indexOf(from);
    const end = rows.indexOf(to);
    if (start < 0 || end < 0) {
        return [];
    }
    return rows.slice(Math.min(start, end), Math.max(start, end) + 1);
}

export function range(
    state: Selection,
    rows: readonly string[],
    target: string,
    rowOf: RowOf,
    childrenOf: ChildrenOf = unknownChildren,
): Selection {
    const anchor = state.anchor;
    const span = anchor === null ? [] : between(rows, anchor, target);
    if (anchor === null || span.length === 0) {
        return toggle(state, target, rowOf(target) ?? bare, childrenOf) ?? { ...state, anchor: target };
    }
    const on = isSelected(state, anchor, rowOf(anchor) ?? bare);
    let next = state;
    for (const id of span) {
        next = set(next, id, rowOf(id) ?? bare, on, childrenOf);
    }
    return next;
}

function holding(pick: SubtreePick, row: SelectionRow): HeldSubtree {
    const size = pick.size === undefined ? (pick.typeId === undefined ? 1 : null) : pick.size;
    return pick.typeId === undefined ? { pageId: pick.pageId, size, row } : { pageId: pick.pageId, typeId: pick.typeId, size, row };
}

function wholly(state: Selection, pick: HeldSubtree): boolean {
    return wholeCovers(state.whole, pick.typeId) || state.subtrees.some((other) => other !== pick && contains(other, pick));
}

export function selectSubtree(
    state: Selection,
    pick: SubtreePick,
    row: SelectionRow = bare,
    childrenOf: ChildrenOf = unknownChildren,
): Selection | null {
    const next = holding(pick, row);
    let base: Selection = {
        ...state,
        except: new Map([...state.except].filter(([id, taken]) => !holds(next, id, taken))),
        outs: state.outs.filter((out) => !contains(next, out)),
    };
    const blocking = base.outs.find((out) => reaches(out, next));
    if (blocking !== undefined) {
        if (blocking.typeId !== next.typeId) {
            return null;
        }
        const open = opened(base, blocking, next.pageId, row, childrenOf);
        if (open === null) {
            return null;
        }
        base = open;
    }
    if (wholly(base, next) || base.subtrees.some((other) => same(other, next))) {
        return settled({ ...base, anchor: pick.pageId });
    }
    return settled({ ...base, subtrees: [...base.subtrees.filter((other) => !contains(next, other)), next], anchor: pick.pageId });
}

export function coveredAbove(state: Selection, id: string, row: SelectionRow, typeId?: string): boolean {
    const scope: HeldSubtree = typeId === undefined ? { pageId: id, size: null, row } : { pageId: id, typeId, size: null, row };
    return state.whole !== null || state.subtrees.some((pick) => !same(pick, scope) && reaches(pick, scope));
}

export function clearSubtree(state: Selection, pick: SubtreePick, row: SelectionRow = bare): Selection {
    const scope: Scope = pick.typeId === undefined ? { pageId: pick.pageId } : { pageId: pick.pageId, typeId: pick.typeId };
    const base: Selection = {
        ...state,
        subtrees: state.subtrees.filter((other) => !contains(scope, other)),
        ids: new Map([...state.ids].filter(([id, held]) => !holds(scope, id, held))),
        except: new Map([...state.except].filter(([id, held]) => !holds(scope, id, held))),
        outs: state.outs.filter((out) => !contains(scope, out)),
        anchor: pick.pageId,
    };
    if (!coveredAbove(base, pick.pageId, row, pick.typeId)) {
        return settled(base);
    }
    const measured = base.subtrees.some((other) => other.typeId === pick.typeId && within(other.pageId, pick.pageId, row));
    const out = holding({ ...pick, size: measured ? (pick.size ?? null) : null }, row);
    return settled({ ...base, outs: [...base.outs, out] });
}

export function stateOf(state: Selection, id: string, row: SelectionRow, scope: TickScope): TickState {
    const typeId = scope.typeId;
    if (state.outs.some((out) => within(out.pageId, id, row) && (out.typeId === undefined || out.typeId === typeId))) {
        return "off";
    }
    const self = isSelected(state, id, row);
    if (scope.leaf) {
        return self ? "on" : "off";
    }
    const full =
        state.subtrees.some((pick) => within(pick.pageId, id, row) && (pick.typeId === undefined || pick.typeId === typeId)) ||
        wholeCovers(state.whole, typeId);
    const inScope = (held: SelectionRow): boolean => typeId === undefined || held.typeId === undefined || held.typeId === typeId;
    const cut =
        [...state.except].some(([taken, held]) => within(id, taken, held) && inScope(held)) ||
        state.outs.some((out) => below(id, out.pageId, out.row));
    if (full) {
        return cut ? "some" : "on";
    }
    const added =
        [...state.ids].some(([picked, held]) => below(id, picked, held)) ||
        state.subtrees.some((pick) => below(id, pick.pageId, pick.row));
    return self || added ? "some" : "off";
}

export function selectAll(state: Selection, filter: PageFacets, seen: number): Selection {
    return settled({ ...state, whole: { kind: "filter", filter, seen }, except: new Map() });
}

export function selectGroup(state: Selection, pageTypeId: string, seen: number): Selection {
    return selectAll(state, { pageTypeId }, seen);
}

export function selectRule(state: Selection, ruleId: string, seen: number): Selection {
    return settled({ ...state, whole: { kind: "rule", ruleId, seen }, except: new Map() });
}

export function dropWhole(state: Selection): Selection {
    return state.whole === null ? state : settled({ ...state, whole: null, except: new Map() });
}

export function clear(): Selection {
    return noSelection;
}

function cleanFilter(filter: PageFacets): PageFacets {
    const out: PageFacets = {};
    for (const [key, value] of Object.entries(filter)) {
        if (value !== undefined && value !== null && value !== "") {
            Object.assign(out, { [key]: value });
        }
    }
    return out;
}

function asked(pick: HeldSubtree): PageSubtree {
    return pick.typeId === undefined ? { pageId: pick.pageId } : { pageId: pick.pageId, typeId: pick.typeId };
}

export function toRequest(siteId: string, state: Selection): PageSelection {
    const out: PageSelection = { siteId };
    if (state.ids.size > 0) {
        out.pageIds = [...state.ids.keys()];
    }
    if (state.subtrees.length > 0) {
        out.subtrees = state.subtrees.map(asked);
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
    if (state.outs.length > 0) {
        out.exceptSubtrees = state.outs.map(asked);
    }
    return out;
}
