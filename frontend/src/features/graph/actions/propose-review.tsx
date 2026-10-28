import type { ChangeEvent, KeyboardEvent, ReactElement } from "react";
import { useEffect, useId, useMemo, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import type { ProposedEntity } from "../../../data/types.js";
import type { SegmentedOption, SelectOption } from "../../../ui/index.js";
import {
    Button,
    Checkbox,
    ChevronRightIcon,
    EmptyState,
    Field,
    Input,
    SearchIcon,
    Segmented,
    Select,
    Stars2Icon,
    Textarea,
    VirtualRows,
    cx,
} from "../../../ui/index.js";
import { clashLabel } from "../labels.js";
import type {
    ParentChoice,
    ProposalEdit,
    ProposalEdits,
    ProposalFilter,
    ProposalSource,
    ProposalTree,
    TreeKey,
    TreeRow,
} from "../model/proposal-tree.js";
import {
    branches,
    changed,
    clashNoName,
    clashesOf,
    everyTickable,
    heldOn,
    keywordLinesOf,
    keywordsOf,
    matchedRows,
    nameOf,
    orphaned,
    parentChoices,
    parentOf,
    proposalFilters,
    proposalTree,
    pruned,
    rangeTicked,
    sharedLabel,
    shownReach,
    spanTicked,
    tickState,
    tickedAll,
    tickedByType,
    tickedTo,
    treeStep,
    turnsOn,
    typeCounts,
    visibleRows,
} from "../model/proposal-tree.js";

const said = copy.graph.ai;

const rowHeight = 38;
const indentPx = 18;

const treeKeys: Readonly<Record<string, TreeKey>> = {
    ArrowUp: "up",
    ArrowDown: "down",
    ArrowLeft: "left",
    ArrowRight: "right",
    Home: "home",
    End: "end",
};

type TickHow = "deep" | "one" | "range";

interface HeldKeys {
    shiftKey: boolean;
    ctrlKey: boolean;
    metaKey: boolean;
}

const noKeys: HeldKeys = Object.freeze({ shiftKey: false, ctrlKey: false, metaKey: false });

function howOf(keys: HeldKeys): TickHow {
    if (keys.shiftKey) {
        return "range";
    }
    return keys.ctrlKey || keys.metaKey ? "one" : "deep";
}

function keysOf(event: ChangeEvent<HTMLInputElement>): HeldKeys {
    const native = event.nativeEvent;
    return native instanceof MouseEvent ? native : noKeys;
}

interface Drag {
    on: boolean;
    deep: boolean;
    kept: ReadonlySet<number>;
    painted: Set<number>;
}

export interface ReviewState {
    kept: ReadonlySet<number>;
    edits: ProposalEdits;
    query: string;
    filter: ProposalFilter;
    matched: ReadonlySet<number> | null;
    collapsed: ReadonlySet<number>;
    focus: number | null;
    anchor: number | null;
    pageTypes: ReadonlyMap<string, string>;
    typeChoice: string | null;
}

export interface ExistingEntity {
    id: string;
    label: string;
}

export interface NamedType {
    id: string;
    name: string;
}


const filterOptions: readonly SegmentedOption<ProposalFilter>[] = proposalFilters.map((value) => ({
    value,
    label: said.filters[value],
}));

function pathOf(proposal: ProposedEntity | undefined): string {
    return proposal?.path ?? "";
}

function choiceValue(choice: ParentChoice): string {
    switch (choice.kind) {
        case "root":
            return "root";
        case "batch":
            return `b:${String(choice.at)}`;
        case "entity":
            return `e:${choice.id}`;
        default:
            return `r:${choice.ref}`;
    }
}

function choiceOf(value: string): ParentChoice {
    if (value.startsWith("b:")) {
        return { kind: "batch", at: Number(value.slice(2)) };
    }
    if (value.startsWith("e:")) {
        return { kind: "entity", id: value.slice(2) };
    }
    if (value.startsWith("r:")) {
        return { kind: "ref", ref: value.slice(2) };
    }
    return { kind: "root" };
}

interface Context {
    proposals: readonly ProposedEntity[];
    edits: ProposalEdits;
    tree: ProposalTree;
    clashes: ReadonlyMap<number, string>;
    kept: ReadonlySet<number>;
    entities: ReadonlyMap<string, string>;
}

function parentLabel(context: Context, at: number): string {
    const choice = parentOf(context.proposals, context.edits, at);
    switch (choice.kind) {
        case "root":
            return said.topLevel;
        case "batch":
            return nameOf(context.proposals, context.edits, choice.at);
        case "entity":
            return context.entities.get(choice.id) ?? (context.proposals[at]?.parent ?? "");
        default:
            return choice.ref;
    }
}

function noteOf(context: Context, at: number): { text: string; tone: string } | null {
    const clash = context.clashes.get(at);
    if (clash !== undefined) {
        const typed = context.edits.get(at)?.name !== undefined;
        return { text: clash === clashNoName && typed ? said.nameNeeded : clashLabel(clash), tone: "text-warn" };
    }
    if (orphaned(context.tree, context.kept, at)) {
        return { text: said.orphan, tone: "text-info" };
    }
    const proposal = context.proposals[at];
    if ((proposal?.existingEntityId ?? "") !== "" && !changed(context.proposals, context.edits, at)) {
        return { text: said.joins, tone: "text-ink-faint" };
    }
    const shown = sharedLabel(context.proposals, context.edits, context.clashes, at);
    if (shown !== null && proposal !== undefined) {
        return { text: said.sharedName(shown, proposal.name.trim()), tone: "text-info" };
    }
    const facts = proposal?.facts ?? [];
    if (facts.length > 0) {
        return { text: said.facts(facts), tone: "text-ink-dim" };
    }
    const keywords = keywordsOf(context.proposals, context.edits, at);
    return keywords.length === 0 ? null : { text: keywords.join(", "), tone: "text-ink-dim" };
}

interface RowProps {
    context: Context;
    row: TreeRow;
    id: string;
    focused: boolean;
    onTick: (at: number, how: TickHow) => void;
    onPress: (at: number, how: TickHow) => void;
    onPaint: (at: number, buttons: number) => void;
    onFold: (at: number) => void;
    onFocus: (at: number) => void;
    onGrab: () => void;
}

function ProposalRow({ context, row, id, focused, onTick, onPress, onPaint, onFold, onFocus, onGrab }: RowProps): ReactElement {
    const { at } = row;
    const proposal = context.proposals[at];
    const name = nameOf(context.proposals, context.edits, at);
    const path = pathOf(proposal);
    const blocked = context.clashes.has(at);
    const state = tickState(context.tree, context.kept, context.clashes, at);
    const note = noteOf(context, at);
    const edited = changed(context.proposals, context.edits, at);
    const shown = name.trim() === "" ? said.unnamed : name;
    const top = !context.tree.parentAt.has(at);
    const under = top && parentOf(context.proposals, context.edits, at).kind !== "root" ? parentLabel(context, at) : "";
    const allBlocked = blocked && state === "off" && !row.branch;
    return (
        <div
            id={id}
            role="treeitem"
            aria-level={row.depth + 1}
            aria-expanded={row.branch ? row.open : undefined}
            aria-checked={state === "some" ? "mixed" : state === "on"}
            aria-disabled={allBlocked ? true : undefined}
            data-proposal={proposal?.name ?? ""}
            data-proposal-page={proposal?.pageId}
            data-proposal-at={at}
            data-proposal-state={blocked ? "blocked" : state}
            aria-current={focused ? true : undefined}
            onMouseDown={(event) => {
                event.preventDefault();
                onGrab();
            }}
            onPointerOver={(event) => {
                onPaint(at, event.buttons);
            }}
            onClick={(event) => {
                if (event.shiftKey || event.ctrlKey || event.metaKey) {
                    onTick(at, howOf(event));
                    return;
                }
                onFocus(at);
            }}
            className={cx(
                "flex cursor-default items-center gap-1.5 border-b border-inset pr-3 select-none",
                focused
                    ? "bg-accent-soft group-focus-visible/tree:shadow-[inset_2px_0_0_var(--color-accent)]"
                    : "hover:bg-raised",
            )}
            style={{ height: rowHeight, paddingLeft: 8 + row.depth * indentPx }}
        >
            {row.branch ? (
                <button
                    type="button"
                    tabIndex={-1}
                    aria-label={row.open ? said.collapseRow : said.expandRow}
                    aria-expanded={row.open}
                    className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-inset hover:text-ink"
                    onClick={(event) => {
                        event.stopPropagation();
                        onFold(at);
                    }}
                >
                    <ChevronRightIcon size={14} className={cx("transition-transform duration-100", row.open && "rotate-90")} />
                </button>
            ) : (
                <span aria-hidden={true} className="w-5 shrink-0" />
            )}
            <span
                onPointerDown={(event) => {
                    if (event.button === 0 && !allBlocked) {
                        onPress(at, howOf(event));
                    }
                }}
                onClick={(event) => {
                    event.stopPropagation();
                    if (event.detail > 0) {
                        event.preventDefault();
                    }
                }}
                className="flex shrink-0 items-center"
            >
                <Checkbox
                    tabIndex={-1}
                    checked={state === "on"}
                    indeterminate={state === "some"}
                    disabled={allBlocked}
                    aria-label={path === "" ? shown : said.keepRow(shown, path)}
                    onChange={(event) => {
                        onTick(at, howOf(keysOf(event)));
                    }}
                />
            </span>
            <div className="flex min-w-0 flex-1 flex-col justify-center gap-0.5">
                <span className="flex min-w-0 items-baseline gap-2">
                    <span
                        title={shown}
                        className={cx(
                            "truncate text-sm",
                            name.trim() === "" ? "text-ink-faint" : blocked || !row.matched ? "text-ink-dim" : "text-ink",
                        )}
                    >
                        {shown}
                    </span>
                    {edited ? <span className="shrink-0 text-2xs text-accent">{said.edited}</span> : null}
                </span>
                {note === null ? null : (
                    <span title={note.text} className={cx("truncate text-2xs", note.tone)}>
                        {note.text}
                    </span>
                )}
            </div>
            <div className="hidden w-[38%] max-w-96 min-w-0 shrink-0 flex-col items-end gap-0.5 @xl:flex">
                <span title={path} className="max-w-full truncate font-mono text-2xs text-ink-dim">
                    {path}
                </span>
                {under === "" ? null : (
                    <span title={under} className="max-w-full truncate text-2xs text-ink-faint">
                        {said.underParent(under)}
                    </span>
                )}
            </div>
        </div>
    );
}

interface InspectorProps {
    context: Context;
    at: number;
    source: ProposalSource;
    existing: readonly ExistingEntity[];
    onEdit: (at: number, edit: ProposalEdit | null) => void;
}

function Inspector({ context, at, source, existing, onEdit }: InspectorProps): ReactElement {
    const proposal = context.proposals[at];
    const edit = context.edits.get(at) ?? {};
    const keywords = keywordsOf(context.proposals, context.edits, at);
    const [draft, setDraft] = useState(keywords.join("\n"));
    const clash = context.clashes.get(at);
    const path = pathOf(proposal);
    const facts = proposal?.facts ?? [];
    const showKeywords = source === "model" || keywords.length > 0;

    const parents: SelectOption<string>[] = useMemo(() => {
        const options: SelectOption<string>[] = [{ value: "root", label: said.parentTop }];
        for (const held of parentChoices(context.tree, context.proposals.length, at)) {
            options.push({
                value: `b:${String(held)}`,
                label: said.parentProposed(nameOf(context.proposals, context.edits, held), pathOf(context.proposals[held])),
            });
        }
        for (const entity of existing) {
            options.push({ value: `e:${entity.id}`, label: said.parentExisting(entity.label) });
        }
        const current = parentOf(context.proposals, context.edits, at);
        if (current.kind === "ref") {
            options.push({ value: choiceValue(current), label: current.ref });
        }
        if (current.kind === "entity" && !existing.some((entity) => entity.id === current.id)) {
            options.push({ value: choiceValue(current), label: parentLabel(context, at) });
        }
        return options;
    }, [context, at, existing]);

    const patch = (next: ProposalEdit): void => {
        onEdit(at, { ...edit, ...next });
    };

    return (
        <div data-proposal-inspector={at} className="flex flex-col gap-3 p-3">
            <div className="flex min-w-0 flex-col gap-0.5">
                <span className="truncate text-sm font-semibold text-ink">
                    {nameOf(context.proposals, context.edits, at).trim() === "" ? said.unnamed : nameOf(context.proposals, context.edits, at)}
                </span>
                {path === "" ? null : <span className="truncate font-mono text-2xs text-ink-dim">{path}</span>}
            </div>
            <Field label={said.nameField} error={clash === undefined ? undefined : noteOf(context, at)?.text}>
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        value={edit.name ?? proposal?.name ?? ""}
                        onChange={(event) => {
                            patch({ name: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field label={said.parentField}>
                {(control) => (
                    <Select
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        value={choiceValue(parentOf(context.proposals, context.edits, at))}
                        options={parents}
                        title={parentLabel(context, at)}
                        onValueChange={(value) => {
                            patch({ parent: choiceOf(value) });
                        }}
                    />
                )}
            </Field>
            {showKeywords ? (
                <Field label={said.keywordsField} hint={said.keywordsEditHint}>
                    {(control) => (
                        <Textarea
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            rows={5}
                            value={draft}
                            onChange={(event) => {
                                setDraft(event.target.value);
                                patch({ keywords: keywordLinesOf(event.target.value) });
                            }}
                        />
                    )}
                </Field>
            ) : null}
            {facts.length === 0 ? null : (
                <div className="flex flex-col gap-1">
                    <span className="text-xs font-medium text-ink-soft">{said.factsField}</span>
                    <ul className="flex flex-col gap-0.5 text-xs text-ink-dim">
                        {facts.map((fact) => (
                            <li key={fact.label} className="flex gap-2">
                                <span className="text-ink-soft">{fact.label}</span>
                                <span className="min-w-0 truncate">{fact.value}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            {context.edits.has(at) ? (
                <Button
                    size="sm"
                    variant="ghost"
                    className="self-start"
                    onClick={() => {
                        onEdit(at, null);
                        setDraft((proposal?.keywords ?? []).map((held) => held.text).join("\n"));
                    }}
                >
                    {said.undoEdits}
                </Button>
            ) : null}
        </div>
    );
}

interface TypeBarProps {
    proposals: readonly ProposedEntity[];
    state: ReviewState;
    kept: ReadonlySet<number>;
    clashes: ReadonlyMap<number, string>;
    types: readonly NamedType[];
    onChange: (patch: Partial<ReviewState>) => void;
}

function TypeBar({ proposals, state, kept, clashes, types, onChange }: TypeBarProps): ReactElement | null {
    const counts = useMemo(() => typeCounts(proposals, state.pageTypes), [proposals, state.pageTypes]);
    const options: SelectOption<string>[] = useMemo(
        () =>
            types
                .filter((type) => counts.has(type.id))
                .map((type) => ({ value: type.id, label: said.typeOption(type.name, counts.get(type.id) ?? 0) })),
        [types, counts],
    );
    if (options.length === 0) {
        return null;
    }
    const chosen = state.typeChoice !== null && counts.has(state.typeChoice) ? state.typeChoice : null;
    const tickType = (on: boolean): void => {
        if (chosen !== null) {
            onChange({ kept: tickedByType(proposals, state.pageTypes, kept, clashes, chosen, on), anchor: null });
        }
    };
    return (
        <div data-proposal-types={true} className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-1.5">
            <span className="text-xs text-ink-soft">{said.byType}</span>
            <div className="w-56 max-w-full">
                <Select
                    size="sm"
                    aria-label={said.byType}
                    placeholder={said.typePlaceholder}
                    value={chosen}
                    options={options}
                    onValueChange={(typeChoice) => {
                        onChange({ typeChoice });
                    }}
                />
            </div>
            <Button size="sm" variant="ghost" disabled={chosen === null} onClick={() => tickType(true)}>
                {said.keepType}
            </Button>
            <Button size="sm" variant="ghost" disabled={chosen === null} onClick={() => tickType(false)}>
                {said.dropType}
            </Button>
        </div>
    );
}

export interface ProposalReviewProps {
    proposals: readonly ProposedEntity[];
    state: ReviewState;
    source: ProposalSource;
    existing: readonly ExistingEntity[];
    types: readonly NamedType[];
    onChange: (patch: Partial<ReviewState>) => void;
}

export function ProposalReview({ proposals, state, source, existing, types, onChange }: ProposalReviewProps): ReactElement {
    const treeId = useId();
    const treeRef = useRef<HTMLDivElement>(null);
    const tree = useMemo(() => proposalTree(proposals, state.edits), [proposals, state.edits]);
    const clashes = useMemo(() => clashesOf(proposals, state.edits), [proposals, state.edits]);
    const kept = useMemo(() => pruned(state.kept, clashes), [state.kept, clashes]);
    const entities = useMemo(() => new Map(existing.map((entity) => [entity.id, entity.label])), [existing]);
    const context: Context = useMemo(
        () => ({ proposals, edits: state.edits, tree, clashes, kept, entities }),
        [proposals, state.edits, tree, clashes, kept, entities],
    );
    const rows = useMemo(() => visibleRows(tree, state.matched, state.collapsed), [tree, state.matched, state.collapsed]);
    const reach = useMemo(() => shownReach(tree, clashes, rows), [tree, clashes, rows]);
    const drag = useRef<Drag | null>(null);
    const unlisten = useRef<(() => void) | null>(null);
    const tickableCount = proposals.length - clashes.size;
    const focus = state.focus !== null && state.focus < proposals.length ? state.focus : null;
    const focusRow = focus === null ? -1 : rows.findIndex((row) => row.at === focus);
    const rowId = (at: number): string => `${treeId}-${String(at)}`;
    const shownRow = (at: number | null): at is number => at !== null && rows.some((row) => row.at === at);

    useEffect(
        () => () => {
            unlisten.current?.();
        },
        [],
    );

    const edit = (at: number, next: ProposalEdit | null): void => {
        const edits = new Map(state.edits);
        if (next === null) {
            edits.delete(at);
        } else {
            edits.set(at, next);
        }
        onChange({ edits });
    };

    const tick = (at: number, how: TickHow): Drag => {
        const deep = how !== "one";
        const on = turnsOn(tree, kept, clashes, at, deep);
        const next = how === "range" ? rangeTicked(tree, kept, clashes, rows, state.anchor, at) : tickedTo(tree, kept, clashes, at, on, deep);
        onChange({ kept: next, anchor: at, focus: at });
        return { on, deep, kept: next, painted: new Set([at]) };
    };

    const endDrag = (): void => {
        drag.current = null;
        unlisten.current?.();
        unlisten.current = null;
    };

    const press = (at: number, how: TickHow): void => {
        endDrag();
        drag.current = tick(at, how);
        const stop = (): void => {
            endDrag();
        };
        document.addEventListener("pointerup", stop);
        document.addEventListener("pointercancel", stop);
        window.addEventListener("blur", stop);
        unlisten.current = () => {
            document.removeEventListener("pointerup", stop);
            document.removeEventListener("pointercancel", stop);
            window.removeEventListener("blur", stop);
        };
    };

    const paint = (at: number, buttons: number): void => {
        const held = drag.current;
        if (held === null || held.painted.has(at)) {
            return;
        }
        if ((buttons & 1) === 0) {
            endDrag();
            return;
        }
        held.painted.add(at);
        held.kept = tickedTo(tree, held.kept, clashes, at, held.on, held.deep);
        onChange({ kept: held.kept });
    };

    const extend = (to: number): void => {
        const from = shownRow(state.anchor) ? state.anchor : shownRow(focus) ? focus : null;
        if (from === null) {
            onChange({ focus: to });
            return;
        }
        onChange({ focus: to, anchor: from, kept: spanTicked(tree, kept, clashes, rows, from, to, heldOn(tree, kept, clashes, from)) });
    };

    const tickShown = (on: boolean): void => {
        onChange({ kept: tickedAll(kept, reach, on), anchor: null });
    };

    const narrow = (query: string, filter: ProposalFilter): void => {
        onChange({ query, filter, matched: matchedRows(proposals, state.edits, kept, query, filter) });
    };

    const fold = (at: number, shut: boolean): void => {
        const collapsed = new Set(state.collapsed);
        if (shut) {
            collapsed.add(at);
        } else {
            collapsed.delete(at);
        }
        onChange({ collapsed });
    };

    const onKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "a") {
            event.preventDefault();
            tickShown(true);
            return;
        }
        const key = treeKeys[event.key];
        if (key !== undefined) {
            event.preventDefault();
            const step = treeStep(rows, tree, state.collapsed, focus, key);
            if (step === null) {
                return;
            }
            if (step.kind !== "focus") {
                fold(step.at, step.kind === "collapse");
                return;
            }
            if (event.shiftKey && (key === "up" || key === "down")) {
                extend(step.at);
                return;
            }
            onChange({ focus: step.at });
            return;
        }
        if (event.key === " " && focus !== null && focusRow >= 0) {
            event.preventDefault();
            tick(focus, howOf(event));
        }
    };

    return (
        <div className="flex min-h-0 flex-1">
            <div className="@container flex min-w-0 flex-1 flex-col">
                <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                    <div className="relative w-64 max-w-full">
                        <SearchIcon
                            size={14}
                            className="pointer-events-none absolute top-1/2 left-2 -translate-y-1/2 text-ink-faint"
                        />
                        <Input
                            type="search"
                            aria-label={said.searchProposals}
                            placeholder={said.searchProposals}
                            value={state.query}
                            className="pl-7"
                            onChange={(event) => {
                                narrow(event.target.value, state.filter);
                            }}
                        />
                    </div>
                    <Segmented
                        label={said.filterLabel}
                        size="sm"
                        value={state.filter}
                        options={filterOptions}
                        onValueChange={(filter) => {
                            narrow(state.query, filter);
                        }}
                    />
                    <span data-proposal-count={kept.size} className="font-mono text-2xs text-ink-faint">
                        {said.tickedOf(kept.size, tickableCount)}
                    </span>
                    {state.matched === null ? null : (
                        <span data-proposal-shown={reach.length} className="flex items-center gap-1">
                            <Button size="sm" variant="ghost" disabled={reach.length === 0} onClick={() => tickShown(true)}>
                                {said.tickShown(reach.length)}
                            </Button>
                            <Button size="sm" variant="ghost" disabled={reach.length === 0} onClick={() => tickShown(false)}>
                                {said.untickShown(reach.length)}
                            </Button>
                        </span>
                    )}
                    <span className="ml-auto flex flex-wrap items-center gap-1">
                        <Button size="sm" variant="ghost" onClick={() => onChange({ collapsed: new Set() })}>
                            {said.expandAll}
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => onChange({ collapsed: branches(tree) })}>
                            {said.collapseAll}
                        </Button>
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => onChange({ kept: everyTickable(proposals.length, clashes), anchor: null })}
                        >
                            {said.keepAll}
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => onChange({ kept: new Set(), anchor: null })}>
                            {said.keepNone}
                        </Button>
                    </span>
                </div>
                <TypeBar proposals={proposals} state={state} kept={kept} clashes={clashes} types={types} onChange={onChange} />
                <div
                    ref={treeRef}
                    role="tree"
                    tabIndex={0}
                    aria-label={said.reviewTitle(proposals.length)}
                    aria-activedescendant={focusRow >= 0 && focus !== null ? rowId(focus) : undefined}
                    onKeyDown={onKeyDown}
                    className="group/tree flex min-h-0 flex-1 flex-col outline-none"
                >
                    {rows.length === 0 ? (
                        <p className="p-3 text-xs text-ink-dim">{said.noMatchProposals}</p>
                    ) : (
                        <VirtualRows
                            count={rows.length}
                            rowHeight={rowHeight}
                            reveal={focusRow >= 0 ? focusRow : undefined}
                            row={(position) => {
                                const row = rows[position];
                                return row === undefined ? null : (
                                    <ProposalRow
                                        context={context}
                                        row={row}
                                        id={rowId(row.at)}
                                        focused={row.at === focus}
                                        onTick={(at, how) => {
                                            tick(at, how);
                                        }}
                                        onPress={press}
                                        onPaint={paint}
                                        onFold={(at) => {
                                            fold(at, !state.collapsed.has(at));
                                        }}
                                        onFocus={(at) => {
                                            onChange({ focus: at });
                                        }}
                                        onGrab={() => {
                                            treeRef.current?.focus({ preventScroll: true });
                                        }}
                                    />
                                );
                            }}
                        />
                    )}
                </div>
            </div>
            <aside aria-label={said.inspectorLabel} className="w-80 shrink-0 overflow-y-auto border-l border-hairline">

                {focus === null ? (
                    <div className="p-4">
                        <EmptyState icon={Stars2Icon} title={said.reviewTitle(proposals.length)} body={said.inspectorEmpty} />
                    </div>
                ) : (
                    <Inspector key={focus} context={context} at={focus} source={source} existing={existing} onEdit={edit} />
                )}
            </aside>
        </div>
    );
}

export interface ProposalConfirmProps {
    written: number;
    joined: number;
}

export function ProposalConfirm({ written, joined }: ProposalConfirmProps): ReactElement {
    return (
        <div className="m-4 flex flex-col gap-1.5 rounded-lg border border-hairline bg-inset p-3" data-proposal-confirm={true}>
            <span className="text-sm font-semibold text-ink">{said.confirmTitle}</span>
            <p className="text-xs leading-snug text-ink-soft">{said.confirmBody(written, joined)}</p>
        </div>
    );
}
