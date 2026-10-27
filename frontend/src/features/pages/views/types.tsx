import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { useTemplates } from "../../../data/hooks/templates.js";
import type { PageSort } from "../../../data/sorts.js";
import type { PageType } from "../../../data/types.js";
import type { SegmentedOption } from "../../../ui/index.js";
import { Checkbox, cx, EmptyState, CategoryIcon, Segmented, SkeletonRows, TuneIcon, toneClasses } from "../../../ui/index.js";
import type { Selection } from "../../../ui/selection.js";
import { selectGroup } from "../../../ui/selection.js";
import { pageTypeOf } from "../../settings/tabs.js";
import { presetIcon, presetTone } from "../../settings/types/labels.js";
import type { EntityIndex } from "../entities.js";
import { PageTable } from "../table.js";
import type { GroupMode } from "./memory.js";
import { groupModes } from "./memory.js";
import type { KnownRow } from "./model.js";
import { groupState, withoutType } from "./model.js";
import type { TypeNaming } from "./naming.js";
import { PagesTree } from "./tree.js";

const said = copy.pages.byType;

const templatePage = 100;

const modeOptions: readonly SegmentedOption<GroupMode>[] = groupModes.map((mode) => ({
    value: mode,
    label: mode === "list" ? copy.pages.views.list : copy.pages.views.tree,
}));

interface GroupRowProps {
    siteId: string;
    type: PageType;
    open: boolean;
    template: string;
    selection: Selection;
    onSelection: (next: Selection) => void;
    onOpen: () => void;
}

function GroupRow({ siteId, type, open, template, selection, onSelection, onOpen }: GroupRowProps): ReactElement {
    const Icon = presetIcon(type.preset);
    const state = groupState(selection, type.id);
    return (
        <li
            data-type-group={type.id}
            data-type-group-state={state}
            className={cx(
                "flex items-start gap-2 border-b border-inset px-3 py-2",
                open ? "bg-accent-soft shadow-[inset_2px_0_0_var(--color-accent)]" : "hover:bg-inset",
            )}
        >
            <span className="mt-0.5 flex shrink-0 items-center">
                <Checkbox
                    aria-label={said.group(type.name)}
                    checked={state === "on"}
                    indeterminate={state === "some"}
                    disabled={type.pages === 0}
                    onChange={() => {
                        onSelection(state === "on" ? withoutType(selection, type.id) : selectGroup(selection, type.id, type.pages));
                    }}
                />
            </span>
            <button
                type="button"
                aria-label={said.open(type.name)}
                aria-pressed={open}
                onClick={onOpen}
                className="flex min-w-0 flex-1 flex-col items-start gap-0.5 rounded-sm text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
                <span className="flex w-full min-w-0 items-center gap-1.5">
                    <Icon size={14} className={cx("shrink-0", toneClasses[presetTone(type.preset)].ink)} />
                    <span className="truncate text-sm font-medium text-ink">{type.name}</span>
                    <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint">{said.pages(type.pages)}</span>
                </span>
                <span className="max-w-full truncate text-2xs text-ink-dim" title={said.template(template)}>
                    {said.template(template)}
                </span>
            </button>
            <Link
                to={pageTypeOf(siteId, type.id)}
                title={said.fieldsOf(type.name)}
                aria-label={said.fieldsOf(type.name)}
                className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-inset hover:text-ink"
            >
                <TuneIcon size={14} />
            </Link>
        </li>
    );
}

export interface PagesByTypeProps {
    siteId: string;
    index: EntityIndex;
    naming: TypeNaming;
    selection: Selection;
    onSelection: (next: Selection) => void;
    openType: string | null;
    mode: GroupMode;
    onOpenType: (typeId: string) => void;
    onMode: (mode: GroupMode) => void;
    onRows: (rows: ReadonlyMap<string, KnownRow>) => void;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
}

export function PagesByType({
    siteId,
    index,
    naming,
    selection,
    onSelection,
    openType,
    mode,
    onOpenType,
    onMode,
    onRows,
    onOpen,
    onUnder,
}: PagesByTypeProps): ReactElement {
    const globals = useTemplates({ scope: "global" }, null, templatePage);
    const locals = useTemplates({ siteId }, null, templatePage);
    const [sort, setSort] = useState<PageSort | null>(null);
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const names = useMemo(
        () => new Map([...flatten(globals.data?.pages), ...flatten(locals.data?.pages)].map((held) => [held.id, held.name] as const)),
        [globals.data, locals.data],
    );
    const shown = naming.types.find((type) => type.id === openType) ?? naming.types.find((type) => type.pages > 0) ?? naming.types[0];

    if (!naming.loaded) {
        return <SkeletonRows rows={8} label={said.loading} className="p-3" />;
    }
    if (shown === undefined) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center p-6">
                <EmptyState icon={CategoryIcon} title={said.empty} />
            </div>
        );
    }

    const templateOf = (type: PageType): string => (type.templateId === null ? said.siteDefault : (names.get(type.templateId) ?? said.siteDefault));

    return (
        <div className="flex min-h-0 flex-1">
            <aside aria-label={said.title} className="w-64 shrink-0 overflow-y-auto border-r border-hairline">
                <ul className="flex flex-col">
                    {naming.types.map((type) => (
                        <GroupRow
                            key={type.id}
                            siteId={siteId}
                            type={type}
                            open={type.id === shown.id}
                            template={templateOf(type)}
                            selection={selection}
                            onSelection={onSelection}
                            onOpen={() => {
                                onOpenType(type.id);
                            }}
                        />
                    ))}
                </ul>
            </aside>
            <section aria-label={shown.name} data-type-open={shown.id} className="flex min-w-0 flex-1 flex-col">
                <div className="flex h-9 shrink-0 items-center gap-3 border-b border-hairline px-3">
                    <span className="truncate text-sm font-medium text-ink">{shown.name}</span>
                    <span className="font-mono text-2xs text-ink-faint">{said.pages(shown.pages)}</span>
                    <span className="ml-auto">
                        <Segmented label={said.mode} size="sm" value={mode} options={modeOptions} onValueChange={onMode} />
                    </span>
                </div>
                {mode === "list" ? (
                    <PageTable
                        key={shown.id}
                        filter={{ siteId, pageTypeId: shown.id }}
                        sort={sort}
                        onSort={setSort}
                        index={index}
                        naming={naming}
                        selection={selection}
                        onSelection={onSelection}
                        onOpen={onOpen}
                        onRows={onRows}
                        empty={<EmptyState icon={CategoryIcon} title={said.none(shown.name)} />}
                        scrollKey={`${siteId}:pages-type:${shown.id}`}
                        label={shown.name}
                    />
                ) : (
                    <PagesTree
                        siteId={siteId}
                        selection={selection}
                        onSelection={onSelection}
                        naming={naming}
                        within={shown.id}
                        scrollKey={`${siteId}:pages-type-tree:${shown.id}`}
                        expanded={expanded}
                        onExpanded={setExpanded}
                        empty={<EmptyState icon={CategoryIcon} title={said.none(shown.name)} />}
                        onRows={onRows}
                        onOpen={onOpen}
                        onUnder={onUnder}
                    />
                )}
            </section>
        </div>
    );
}
