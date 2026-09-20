import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useEntities } from "../../data/hooks/graph.js";
import { usePages } from "../../data/hooks/pages.js";
import { useRuns } from "../../data/hooks/runs.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import { pushToast } from "../../data/toasts.js";
import {
    AddIcon,
    BoltIcon,
    DescriptionIcon,
    HubIcon,
    Kbd,
    PlayCircleIcon,
    SearchIcon,
    SmartToyIcon,
    SyncIcon,
    cx,
} from "../../ui/index.js";
import type { IconComponent } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { rankBy } from "./rank.js";
import { closePalette, usePaletteOpen } from "./state.js";

const debounceMs = 180;
const perSection = 6;

interface Row {
    key: string;
    label: string;
    kind: string;
    Icon: IconComponent;
    run: () => void;
}

interface Section {
    key: string;
    label: string;
    rows: readonly Row[];
}

function useDebounced(value: string): string {
    const [settled, setSettled] = useState(value);
    useEffect(() => {
        const timer = window.setTimeout(() => {
            setSettled(value);
        }, debounceMs);
        return () => {
            window.clearTimeout(timer);
        };
    }, [value]);
    return settled;
}

export interface Destination {
    key: string;
    label: string;
    to: string;
    Icon: IconComponent;
}

export interface CommandPaletteProps {
    siteId: string | null;
    destinations: readonly Destination[];
}

export function CommandPalette({ siteId, destinations }: CommandPaletteProps): ReactElement | null {
    const open = usePaletteOpen();
    if (!open) {
        return null;
    }
    return <PaletteDialog siteId={siteId} destinations={destinations} />;
}

function PaletteDialog({ siteId, destinations }: CommandPaletteProps): ReactElement {
    const navigate = useNavigate();
    const sync = useSyncSite();
    const [query, setQuery] = useState("");
    const [active, setActive] = useState(0);
    const settled = useDebounced(query.trim());
    const listRef = useRef<HTMLDivElement | null>(null);

    const entities = useEntities(
        { siteId: siteId ?? "", namePrefix: settled },
        { field: "name", desc: false },
        perSection,
    );
    const pages = usePages({ siteId: siteId ?? "", pathPrefix: settled }, { field: "path", desc: false }, perSection);
    const runs = useRuns(siteId === null ? {} : { siteId }, null, perSection);

    const go = (to: string): (() => void) => {
        return () => {
            closePalette();
            void navigate(to);
        };
    };

    const sections = useMemo<readonly Section[]>(() => {
        const said = copy.palette;
        const screens: Row[] = rankBy(settled, destinations, (entry) => entry.label, perSection).map(
            (entry) => ({
                key: `screen:${entry.key}`,
                label: entry.label,
                kind: said.kinds.screen,
                Icon: entry.Icon,
                run: go(entry.to),
            }),
        );

        const actions: Row[] = [];
        if (siteId !== null) {
            actions.push({
                key: "action:run",
                label: said.newRun,
                kind: said.kinds.action,
                Icon: BoltIcon,
                run: go(`/s/${siteId}/runs?action=new`),
            });
            actions.push({
                key: "action:entity",
                label: said.newEntity,
                kind: said.kinds.action,
                Icon: AddIcon,
                run: go(`/s/${siteId}/graph?action=new`),
            });
            actions.push({
                key: "action:sync",
                label: said.syncSite,
                kind: said.kinds.action,
                Icon: SyncIcon,
                run: () => {
                    closePalette();
                    sync.mutate(
                        { siteId },
                        {
                            onSuccess: () => {
                                pushToast("info", said.syncing);
                            },
                        },
                    );
                },
            });
        }
        actions.push({
            key: "action:agent",
            label: said.askAgent,
            kind: said.kinds.action,
            Icon: SmartToyIcon,
            run: () => {
                closePalette();
                askAgent("");
            },
        });

        const entityRows: Row[] = flatten(entities.data?.pages).map((entity) => ({
            key: `entity:${entity.id}`,
            label: entity.name,
            kind: said.kinds.entity,
            Icon: HubIcon,
            run: go(`/s/${entity.siteId}/graph/${entity.id}`),
        }));

        const pageRows: Row[] = flatten(pages.data?.pages).map((page) => ({
            key: `page:${page.id}`,
            label: page.path,
            kind: said.kinds.page,
            Icon: DescriptionIcon,
            run: go(`/s/${page.siteId}/pages/${page.id}`),
        }));

        const runRows: Row[] = rankBy(
            settled,
            flatten(runs.data?.pages),
            (held) => `${held.kind} ${held.status}`,
            perSection,
        ).map((held) => ({
            key: `run:${held.id}`,
            label: said.run(held.id.slice(0, 8), held.kind),
            kind: said.kinds.run,
            Icon: PlayCircleIcon,
            run: go(`/s/${held.siteId}/runs/${held.id}`),
        }));

        return [
            { key: "goTo", label: said.goTo, rows: screens },
            { key: "actions", label: said.actions, rows: rankBy(settled, actions, (row) => row.label, perSection) },
            { key: "entities", label: said.entities, rows: entityRows },
            { key: "pages", label: said.pages, rows: pageRows },
            { key: "runs", label: said.runs, rows: runRows },
        ].filter((section) => section.rows.length > 0);
    }, [settled, siteId, destinations, entities.data, pages.data, runs.data, navigate, sync]);

    const rows = useMemo(() => sections.flatMap((section) => section.rows), [sections]);
    const at = rows.length === 0 ? -1 : Math.min(active, rows.length - 1);

    useEffect(() => {
        setActive(0);
    }, [settled]);

    useEffect(() => {
        listRef.current?.querySelector('[data-active="true"]')?.scrollIntoView({ block: "nearest" });
    }, [at]);

    return (
        <RadixDialog.Root
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    closePalette();
                }
            }}
        >
            <RadixDialog.Portal>
                <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/55 data-[state=open]:animate-fade-in" />
                <RadixDialog.Content
                    aria-label={copy.palette.title}
                    className="data-[state=open]:animate-fade-in fixed top-24 left-1/2 z-50 flex w-[min(36rem,calc(100vw-3rem))] -translate-x-1/2 flex-col overflow-hidden rounded-lg border border-edge bg-inset"
                    onKeyDown={(event) => {
                        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                            event.preventDefault();
                            if (rows.length > 0) {
                                const delta = event.key === "ArrowDown" ? 1 : rows.length - 1;
                                setActive((held) => (Math.min(held, rows.length - 1) + delta) % rows.length);
                            }
                            return;
                        }
                        if (event.key === "Enter" && at >= 0) {
                            event.preventDefault();
                            rows[at].run();
                        }
                    }}
                >
                    <RadixDialog.Title className="sr-only">{copy.palette.title}</RadixDialog.Title>
                    <div className="flex h-10 shrink-0 items-center gap-2 border-b border-hairline px-3">
                        <SearchIcon size={16} className="shrink-0 text-ink-dim" />
                        <input
                            data-palette-query={true}
                            autoFocus={true}
                            value={query}
                            placeholder={copy.palette.placeholder}
                            aria-label={copy.palette.title}
                            onChange={(event) => {
                                setQuery(event.target.value);
                            }}
                            className="min-w-0 flex-1 bg-transparent text-base text-ink outline-none placeholder:text-ink-faint"
                        />
                    </div>
                    <div ref={listRef} className="max-h-96 min-h-0 flex-1 overflow-auto py-1">
                        {rows.length === 0 ? (
                            <p className="px-3 py-3 text-xs text-ink-dim">
                                {siteId === null ? copy.palette.needSite : copy.palette.nothing}
                            </p>
                        ) : (
                            sections.map((section) => (
                                <div key={section.key}>
                                    <p className="px-3 pt-2 pb-1 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                                        {section.label}
                                    </p>
                                    {section.rows.map((row) => {
                                        const picked = rows[at]?.key === row.key;
                                        return (
                                            <button
                                                key={row.key}
                                                type="button"
                                                tabIndex={-1}
                                                data-active={picked}
                                                onMouseEnter={() => {
                                                    setActive(rows.findIndex((held) => held.key === row.key));
                                                }}
                                                onClick={row.run}
                                                className={cx(
                                                    "flex h-7 w-full items-center gap-2 px-3 text-left",
                                                    picked ? "bg-accent-soft text-ink" : "text-ink-soft",
                                                )}
                                            >
                                                <row.Icon size={14} className="shrink-0 text-ink-dim" />
                                                <span className="min-w-0 flex-1 truncate text-sm">{row.label}</span>
                                                <span className="shrink-0 font-mono text-2xs text-ink-faint">
                                                    {row.kind}
                                                </span>
                                            </button>
                                        );
                                    })}
                                </div>
                            ))
                        )}
                    </div>
                    <div className="flex shrink-0 items-center gap-3 border-t border-hairline px-3 py-1.5 text-2xs text-ink-faint">
                        <span className="flex items-center gap-1">
                            <Kbd keys={["↑", "↓"]} />
                            {copy.palette.navigate}
                        </span>
                        <span className="flex items-center gap-1">
                            <Kbd keys={["↵"]} />
                            {copy.palette.open}
                        </span>
                        <span className="flex items-center gap-1">
                            <Kbd keys={["Esc"]} />
                            {copy.palette.close}
                        </span>
                    </div>
                </RadixDialog.Content>
            </RadixDialog.Portal>
        </RadixDialog.Root>
    );
}
