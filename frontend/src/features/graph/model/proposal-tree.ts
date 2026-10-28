import type { ProposedEntity } from "../../../data/types.js";
import { between } from "../../../ui/selection.js";

export type ParentChoice =
    | { kind: "root" }
    | { kind: "batch"; at: number }
    | { kind: "entity"; id: string }
    | { kind: "ref"; ref: string };

export interface ProposalEdit {
    name?: string;
    parent?: ParentChoice;
    keywords?: readonly string[];
}

export type ProposalEdits = ReadonlyMap<number, ProposalEdit>;

export type ProposalSource = "model" | "site";

export type TickState = "on" | "off" | "some";

export type ProposalFilter = "all" | "ticked" | "unticked" | "clash";

export const proposalFilters: readonly ProposalFilter[] = ["all", "ticked", "unticked", "clash"];

export const clashNoName = "no_name";
export const clashInBatch = "name_in_batch";
export const clashCarried = "name_carried";

const rootChoice: ParentChoice = Object.freeze({ kind: "root" });

export function entityKey(text: string): string {
    return text.trim().replace(/[A-Z]/g, (letter) => letter.toLowerCase());
}

function pathOf(proposal: ProposedEntity | undefined): string {
    return (proposal?.path ?? "").trim();
}

export function originalParent(proposals: readonly ProposedEntity[], at: number): ParentChoice {
    const proposal = proposals[at];
    if (proposal === undefined) {
        return rootChoice;
    }
    const entityId = (proposal.parentEntityId ?? "").trim();
    if (entityId !== "") {
        return { kind: "entity", id: entityId };
    }
    const ref = (proposal.parent ?? "").trim();
    if (ref === "") {
        return rootChoice;
    }
    const byPath = proposals.findIndex((other, index) => index !== at && pathOf(other) === ref);
    if (byPath >= 0) {
        return { kind: "batch", at: byPath };
    }
    const byName = proposals.findIndex(
        (other, index) => index !== at && pathOf(other) === "" && entityKey(other.name) === entityKey(ref),
    );
    if (byName >= 0) {
        return { kind: "batch", at: byName };
    }
    return { kind: "ref", ref };
}

export function sameChoice(left: ParentChoice, right: ParentChoice): boolean {
    switch (left.kind) {
        case "root":
            return right.kind === "root";
        case "batch":
            return right.kind === "batch" && right.at === left.at;
        case "entity":
            return right.kind === "entity" && right.id === left.id;
        default:
            return right.kind === "ref" && right.ref === left.ref;
    }
}

export function parentOf(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number): ParentChoice {
    return edits.get(at)?.parent ?? originalParent(proposals, at);
}

export function nameOf(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number): string {
    const edited = edits.get(at)?.name;
    if (edited !== undefined) {
        return edited;
    }
    const proposal = proposals[at];
    if (proposal === undefined) {
        return "";
    }
    return (proposal.label ?? "") === "" ? proposal.name : (proposal.label ?? "");
}

function renamed(proposal: ProposedEntity, edit: ProposalEdit | undefined): boolean {
    return edit?.name !== undefined && edit.name.trim() !== proposal.name.trim();
}

export function moved(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number): boolean {
    const choice = edits.get(at)?.parent;
    return choice !== undefined && !sameChoice(choice, originalParent(proposals, at));
}

function rekeyed(proposal: ProposedEntity, edit: ProposalEdit | undefined): boolean {
    if (edit?.keywords === undefined) {
        return false;
    }
    const before = (proposal.keywords ?? []).map((held) => held.text.trim());
    return before.length !== edit.keywords.length || before.some((text, index) => text !== edit.keywords?.[index]);
}

export function changed(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number): boolean {
    const proposal = proposals[at];
    if (proposal === undefined) {
        return false;
    }
    const edit = edits.get(at);
    return renamed(proposal, edit) || moved(proposals, edits, at) || rekeyed(proposal, edit);
}

export interface ProposalTree {
    roots: readonly number[];
    children: ReadonlyMap<number, readonly number[]>;
    parentAt: ReadonlyMap<number, number>;
}

function loops(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number, parent: number): boolean {
    const seen = new Set<number>([at]);
    for (let held: number | null = parent; held !== null; ) {
        if (seen.has(held)) {
            return true;
        }
        seen.add(held);
        const above = parentOf(proposals, edits, held);
        held = above.kind === "batch" ? above.at : null;
    }
    return false;
}

export function proposalTree(proposals: readonly ProposedEntity[], edits: ProposalEdits): ProposalTree {
    const roots: number[] = [];
    const children = new Map<number, number[]>();
    const parentAt = new Map<number, number>();
    proposals.forEach((_, at) => {
        const choice = parentOf(proposals, edits, at);
        if (choice.kind !== "batch" || choice.at >= proposals.length || loops(proposals, edits, at, choice.at)) {
            roots.push(at);
            return;
        }
        parentAt.set(at, choice.at);
        const held = children.get(choice.at);
        if (held === undefined) {
            children.set(choice.at, [at]);
        } else {
            held.push(at);
        }
    });
    return { roots, children, parentAt };
}

export function subtree(tree: ProposalTree, at: number): number[] {
    const out: number[] = [];
    const stack = [at];
    while (stack.length > 0) {
        const held = stack.pop();
        if (held === undefined) {
            break;
        }
        out.push(held);
        const below = tree.children.get(held) ?? [];
        for (let index = below.length - 1; index >= 0; index -= 1) {
            const child = below[index];
            if (child !== undefined) {
                stack.push(child);
            }
        }
    }
    return out;
}

function scopeKey(choice: ParentChoice): string {
    switch (choice.kind) {
        case "root":
            return "";
        case "batch":
            return `b${String(choice.at)}`;
        case "entity":
            return `e${choice.id}`;
        default:
            return `r${entityKey(choice.ref)}`;
    }
}

export function clashesOf(proposals: readonly ProposedEntity[], edits: ProposalEdits): ReadonlyMap<number, string> {
    const out = new Map<number, string>();
    const taken = new Set<string>();
    proposals.forEach((proposal, at) => {
        const edit = edits.get(at);
        const name = edit?.name ?? proposal.name;
        if (name.trim() === "") {
            out.set(at, clashNoName);
            return;
        }
        if (proposal.clash === clashCarried && !renamed(proposal, edit) && !moved(proposals, edits, at)) {
            out.set(at, clashCarried);
            return;
        }
        const key = `${scopeKey(parentOf(proposals, edits, at))}\u0000${entityKey(name)}`;
        if (taken.has(key)) {
            out.set(at, clashInBatch);
            return;
        }
        taken.add(key);
    });
    return out;
}

export function tickable(clashes: ReadonlyMap<number, string>, at: number): boolean {
    return !clashes.has(at);
}

export function everyTickable(count: number, clashes: ReadonlyMap<number, string>): ReadonlySet<number> {
    const out = new Set<number>();
    for (let at = 0; at < count; at += 1) {
        if (tickable(clashes, at)) {
            out.add(at);
        }
    }
    return out;
}

export function pruned(kept: ReadonlySet<number>, clashes: ReadonlyMap<number, string>): ReadonlySet<number> {
    if ([...kept].every((at) => tickable(clashes, at))) {
        return kept;
    }
    return new Set([...kept].filter((at) => tickable(clashes, at)));
}

export function tickState(
    tree: ProposalTree,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    at: number,
): TickState {
    const open = subtree(tree, at).filter((held) => tickable(clashes, held));
    const on = open.filter((held) => kept.has(held)).length;
    if (on === 0) {
        return "off";
    }
    return on === open.length ? "on" : "some";
}

function paint(
    next: Set<number>,
    tree: ProposalTree,
    clashes: ReadonlyMap<number, string>,
    at: number,
    on: boolean,
    deep: boolean,
): void {
    for (const held of deep ? subtree(tree, at) : [at]) {
        if (!tickable(clashes, held)) {
            continue;
        }
        if (on) {
            next.add(held);
        } else {
            next.delete(held);
        }
    }
}

export function turnsOn(
    tree: ProposalTree,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    at: number,
    deep: boolean,
): boolean {
    return deep ? tickState(tree, kept, clashes, at) !== "on" : !kept.has(at);
}

export function heldOn(tree: ProposalTree, kept: ReadonlySet<number>, clashes: ReadonlyMap<number, string>, at: number): boolean {
    return tickable(clashes, at) ? kept.has(at) : tickState(tree, kept, clashes, at) === "on";
}

export function tickedTo(
    tree: ProposalTree,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    at: number,
    on: boolean,
    deep: boolean,
): ReadonlySet<number> {
    const next = new Set(kept);
    paint(next, tree, clashes, at, on, deep);
    return next;
}

export function spanTicked(
    tree: ProposalTree,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    rows: readonly TreeRow[],
    from: number,
    to: number,
    on: boolean,
): ReadonlySet<number> {
    const span = between(
        rows.map((row) => row.at),
        from,
        to,
    );
    const next = new Set(kept);
    for (const held of span.length === 0 ? [to] : span) {
        paint(next, tree, clashes, held, on, true);
    }
    return next;
}

export function rangeTicked(
    tree: ProposalTree,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    rows: readonly TreeRow[],
    anchor: number | null,
    target: number,
): ReadonlySet<number> {
    return spanTicked(tree, kept, clashes, rows, anchor ?? target, target, turnsOn(tree, kept, clashes, target, true));
}

export function shownReach(tree: ProposalTree, clashes: ReadonlyMap<number, string>, rows: readonly TreeRow[]): readonly number[] {
    const out = new Set<number>();
    for (const row of rows) {
        if (!row.matched) {
            continue;
        }
        for (const held of row.branch && !row.open ? subtree(tree, row.at) : [row.at]) {
            if (tickable(clashes, held)) {
                out.add(held);
            }
        }
    }
    return [...out];
}

export function tickedAll(kept: ReadonlySet<number>, ats: readonly number[], on: boolean): ReadonlySet<number> {
    const next = new Set(kept);
    for (const at of ats) {
        if (on) {
            next.add(at);
        } else {
            next.delete(at);
        }
    }
    return next;
}

export function sourceTypesOf(held: Readonly<Record<string, string | undefined>> | null | undefined): ReadonlyMap<string, string> {
    const out = new Map<string, string>();
    for (const [pageId, typeId] of Object.entries(held ?? {})) {
        if (typeId !== undefined && typeId !== "") {
            out.set(pageId, typeId);
        }
    }
    return out;
}

function typeAt(proposals: readonly ProposedEntity[], pageTypes: ReadonlyMap<string, string>, at: number): string | undefined {
    const pageId = proposals[at]?.pageId ?? "";
    return pageId === "" ? undefined : pageTypes.get(pageId);
}

export function typeCounts(proposals: readonly ProposedEntity[], pageTypes: ReadonlyMap<string, string>): ReadonlyMap<string, number> {
    const out = new Map<string, number>();
    proposals.forEach((_, at) => {
        const typeId = typeAt(proposals, pageTypes, at);
        if (typeId !== undefined) {
            out.set(typeId, (out.get(typeId) ?? 0) + 1);
        }
    });
    return out;
}

export function tickedByType(
    proposals: readonly ProposedEntity[],
    pageTypes: ReadonlyMap<string, string>,
    kept: ReadonlySet<number>,
    clashes: ReadonlyMap<number, string>,
    typeId: string,
    on: boolean,
): ReadonlySet<number> {
    const next = new Set(kept);
    proposals.forEach((_, at) => {
        if (typeAt(proposals, pageTypes, at) !== typeId || !tickable(clashes, at)) {
            return;
        }
        if (on) {
            next.add(at);
        } else {
            next.delete(at);
        }
    });
    return next;
}

export function sharedLabel(
    proposals: readonly ProposedEntity[],
    edits: ProposalEdits,
    clashes: ReadonlyMap<number, string>,
    at: number,
): string | null {
    const proposal = proposals[at];
    if (proposal === undefined || clashes.has(at) || renamed(proposal, edits.get(at)) || moved(proposals, edits, at)) {
        return null;
    }
    const label = (proposal.label ?? "").trim();
    return label === "" || label === proposal.name.trim() ? null : label;
}

export type TreeKey = "up" | "down" | "left" | "right" | "home" | "end";

export type TreeStep = { kind: "focus"; at: number } | { kind: "collapse"; at: number } | { kind: "expand"; at: number };

export function treeStep(
    rows: readonly TreeRow[],
    tree: ProposalTree,
    collapsed: ReadonlySet<number>,
    focus: number | null,
    key: TreeKey,
): TreeStep | null {
    const first = rows[0];
    const last = rows[rows.length - 1];
    if (first === undefined || last === undefined) {
        return null;
    }
    const position = focus === null ? -1 : rows.findIndex((row) => row.at === focus);
    const row = rows[position];
    if (key === "home" || (row === undefined && (key === "up" || key === "down"))) {
        return { kind: "focus", at: first.at };
    }
    if (key === "end") {
        return { kind: "focus", at: last.at };
    }
    if (row === undefined) {
        return null;
    }
    switch (key) {
        case "down": {
            const next = rows[position + 1];
            return next === undefined ? null : { kind: "focus", at: next.at };
        }
        case "up": {
            const next = rows[position - 1];
            return next === undefined ? null : { kind: "focus", at: next.at };
        }
        case "left": {
            if (row.branch && row.open && !collapsed.has(row.at)) {
                return { kind: "collapse", at: row.at };
            }
            const parent = tree.parentAt.get(row.at);
            return parent === undefined || !rows.some((held) => held.at === parent) ? null : { kind: "focus", at: parent };
        }
        default: {
            if (!row.branch) {
                return null;
            }
            if (!row.open) {
                return { kind: "expand", at: row.at };
            }
            const child = rows[position + 1];
            return child === undefined || child.depth <= row.depth ? null : { kind: "focus", at: child.at };
        }
    }
}

export function orphaned(tree: ProposalTree, kept: ReadonlySet<number>, at: number): boolean {
    const parent = tree.parentAt.get(at);
    return kept.has(at) && parent !== undefined && !kept.has(parent);
}

export function parentChoices(tree: ProposalTree, count: number, at: number): number[] {
    const inside = new Set(subtree(tree, at));
    const out: number[] = [];
    for (let held = 0; held < count; held += 1) {
        if (!inside.has(held)) {
            out.push(held);
        }
    }
    return out;
}

export function keywordsOf(proposals: readonly ProposedEntity[], edits: ProposalEdits, at: number): readonly string[] {
    return edits.get(at)?.keywords ?? (proposals[at]?.keywords ?? []).map((held) => held.text);
}

export function keywordLinesOf(text: string): string[] {
    const seen = new Set<string>();
    const out: string[] = [];
    for (const raw of text.split(/\r?\n/)) {
        const line = raw.trim();
        if (line === "" || seen.has(line)) {
            continue;
        }
        seen.add(line);
        out.push(line);
    }
    return out;
}

export interface SendContext {
    source: ProposalSource;
    entityName: (id: string) => string | undefined;
}

function placed(
    proposals: readonly ProposedEntity[],
    edits: ProposalEdits,
    sent: ReadonlySet<number>,
    at: number,
    context: SendContext,
): Pick<ProposedEntity, "parent" | "parentEntityId"> {
    const choice = parentOf(proposals, edits, at);
    switch (choice.kind) {
        case "root":
            return { parent: "", parentEntityId: "" };
        case "batch": {
            if (!sent.has(choice.at)) {
                return { parent: "", parentEntityId: "" };
            }
            const target = proposals[choice.at];
            const path = pathOf(target);
            if (context.source === "site" || path !== "") {
                return { parent: path, parentEntityId: "" };
            }
            return { parent: (edits.get(choice.at)?.name ?? target?.name ?? "").trim(), parentEntityId: "" };
        }
        case "entity":
            if (context.source === "site") {
                return { parent: "", parentEntityId: choice.id };
            }
            return { parent: context.entityName(choice.id) ?? "", parentEntityId: choice.id };
        default:
            return { parent: choice.ref, parentEntityId: "" };
    }
}

export function sentProposals(
    proposals: readonly ProposedEntity[],
    edits: ProposalEdits,
    kept: ReadonlySet<number>,
    context: SendContext,
): ProposedEntity[] {
    const clashes = clashesOf(proposals, edits);
    const sent = new Set([...kept].filter((at) => at < proposals.length && tickable(clashes, at)));
    const out: ProposedEntity[] = [];
    proposals.forEach((proposal, at) => {
        if (!sent.has(at)) {
            return;
        }
        const choice = parentOf(proposals, edits, at);
        const parentGone = choice.kind === "batch" && !sent.has(choice.at);
        const parentRenamed =
            choice.kind === "batch" &&
            context.source === "model" &&
            pathOf(proposals[choice.at]) === "" &&
            changed(proposals, edits, choice.at);
        if (!changed(proposals, edits, at) && !parentGone && !parentRenamed) {
            out.push(proposal);
            return;
        }
        const edit = edits.get(at);
        const next: ProposedEntity = { ...proposal, ...placed(proposals, edits, sent, at, context) };
        if (renamed(proposal, edit) || moved(proposals, edits, at)) {
            const name = (edit?.name ?? proposal.name).trim();
            next.name = name;
            next.label = name;
            delete next.existingEntityId;
            delete next.clash;
        }
        if (rekeyed(proposal, edit)) {
            const volumes = new Map((proposal.keywords ?? []).map((held) => [held.text.trim(), held.volume]));
            next.keywords = (edit?.keywords ?? []).map((text) => {
                const volume = volumes.get(text);
                return volume === undefined || volume === null ? { text } : { text, volume };
            });
        }
        out.push(next);
    });
    return out;
}

export interface TreeRow {
    at: number;
    depth: number;
    branch: boolean;
    open: boolean;
    matched: boolean;
}

export function matchesQuery(name: string, path: string, query: string): boolean {
    const needle = query.trim().toLowerCase();
    return needle === "" || name.toLowerCase().includes(needle) || path.toLowerCase().includes(needle);
}

function passes(filter: ProposalFilter, kept: ReadonlySet<number>, clashes: ReadonlyMap<number, string>, at: number): boolean {
    switch (filter) {
        case "ticked":
            return kept.has(at);
        case "unticked":
            return !kept.has(at) && tickable(clashes, at);
        case "clash":
            return !tickable(clashes, at);
        default:
            return true;
    }
}

export function matchedRows(
    proposals: readonly ProposedEntity[],
    edits: ProposalEdits,
    kept: ReadonlySet<number>,
    query: string,
    filter: ProposalFilter,
): ReadonlySet<number> | null {
    if (query.trim() === "" && filter === "all") {
        return null;
    }
    const clashes = clashesOf(proposals, edits);
    const out = new Set<number>();
    proposals.forEach((proposal, at) => {
        if (passes(filter, kept, clashes, at) && matchesQuery(nameOf(proposals, edits, at), pathOf(proposal), query)) {
            out.add(at);
        }
    });
    return out;
}

export function visibleRows(tree: ProposalTree, matched: ReadonlySet<number> | null, collapsed: ReadonlySet<number>): TreeRow[] {
    const narrowed = matched !== null;
    const wanted = new Set<number>();
    for (const at of matched ?? []) {
        for (let held: number | undefined = at; held !== undefined; held = tree.parentAt.get(held)) {
            if (wanted.has(held)) {
                break;
            }
            wanted.add(held);
        }
    }
    const out: TreeRow[] = [];
    const walk = (at: number, depth: number): void => {
        if (narrowed && !wanted.has(at)) {
            return;
        }
        const below = (tree.children.get(at) ?? []).filter((child) => !narrowed || wanted.has(child));
        const open = narrowed || !collapsed.has(at);
        out.push({ at, depth, branch: below.length > 0, open, matched: matched === null || matched.has(at) });
        if (open) {
            for (const child of below) {
                walk(child, depth + 1);
            }
        }
    };
    for (const root of tree.roots) {
        walk(root, 0);
    }
    return out;
}

export function branches(tree: ProposalTree): ReadonlySet<number> {
    return new Set(tree.children.keys());
}
