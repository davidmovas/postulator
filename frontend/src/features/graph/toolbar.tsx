import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../copy/index.js";
import { AddIcon, AddLinkIcon, Button, CloseIcon, cx, HubIcon, PolylineIcon, SearchIcon, TableRowsIcon, toneClasses } from "../../ui/index.js";
import type { IconComponent } from "../../ui/index.js";
import { entityIcon, kindTone } from "./labels.js";
import type { GraphIndex } from "./model/index.js";
import type { GraphView } from "./model/params.js";
import { rank } from "./model/search.js";

const hitLimit = 8;

interface ViewButtonProps {
    view: GraphView;
    current: GraphView;
    label: string;
    icon: IconComponent;
    onSelect: (view: GraphView) => void;
}

function ViewButton({ view, current, label, icon: Icon, onSelect }: ViewButtonProps): ReactElement {
    const active = view === current;
    return (
        <button
            type="button"
            aria-pressed={active}
            onClick={() => {
                onSelect(view);
            }}
            className={cx(
                "inline-flex h-6 items-center gap-1 rounded-md px-2 text-xs font-medium transition-colors duration-100",
                active ? "bg-raised text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
            )}
        >
            <Icon size={14} className="shrink-0" />
            {label}
        </button>
    );
}

function Key({ children }: { children: string }): ReactElement {
    return <kbd className="ml-1 rounded-sm border border-edge bg-raised px-1 font-mono text-2xs text-ink-dim">{children}</kbd>;
}

export interface SearchBoxProps {
    index: GraphIndex;
    onPick: (id: string) => void;
    focusKey: number;
}

export function SearchBox({ index, onPick, focusKey }: SearchBoxProps): ReactElement {
    const [query, setQuery] = useState("");
    const [open, setOpen] = useState(false);
    const [active, setActive] = useState(0);
    const input = useRef<HTMLInputElement>(null);
    const hits = useMemo(() => rank(index.entities, query, hitLimit), [index, query]);

    useEffect(() => {
        if (focusKey > 0) {
            input.current?.focus();
            input.current?.select();
        }
    }, [focusKey]);

    const pick = (id: string): void => {
        onPick(id);
        setOpen(false);
        setQuery("");
    };

    const onKeyDown = (event: KeyboardEvent<HTMLInputElement>): void => {
        if (event.key === "ArrowDown") {
            event.preventDefault();
            setActive((held) => Math.min(held + 1, Math.max(hits.length - 1, 0)));
        } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActive((held) => Math.max(held - 1, 0));
        } else if (event.key === "Enter") {
            const hit = hits[active];
            if (hit !== undefined) {
                pick(hit.id);
            }
        } else if (event.key === "Escape") {
            setOpen(false);
            input.current?.blur();
        }
    };

    return (
        <div className="relative">
            <label className="flex h-6 w-56 items-center gap-1 rounded-md border border-hairline bg-inset px-2 focus-within:border-accent">
                <SearchIcon size={14} className="shrink-0 text-ink-faint" />
                <span className="sr-only">{copy.graph.search.label}</span>
                <input
                    ref={input}
                    type="search"
                    value={query}
                    placeholder={copy.graph.search.placeholder}
                    className="min-w-0 flex-1 bg-transparent text-xs text-ink outline-none placeholder:text-ink-faint"
                    onChange={(event) => {
                        setQuery(event.target.value);
                        setActive(0);
                        setOpen(true);
                    }}
                    onFocus={() => {
                        setOpen(true);
                    }}
                    onBlur={() => {
                        window.setTimeout(() => {
                            setOpen(false);
                        }, 120);
                    }}
                    onKeyDown={onKeyDown}
                />
            </label>
            {open && query.trim() !== "" ? (
                <ul role="listbox" className="absolute top-7 left-0 z-20 w-72 overflow-hidden rounded-md border border-edge bg-raised py-1">
                    {hits.length === 0 ? (
                        <li className="px-2 py-1 text-xs text-ink-dim">{copy.graph.search.noHits}</li>
                    ) : (
                        hits.map((hit, position) => {
                            const held = index.byId.get(hit.id);
                            const Icon = entityIcon(held?.kind ?? "");
                            return (
                                <li
                                    key={hit.id}
                                    role="option"
                                    aria-selected={position === active}
                                    className={cx(
                                        "flex cursor-pointer items-center gap-2 px-2 py-1 text-xs",
                                        position === active ? "bg-accent-soft text-ink" : "text-ink-soft",
                                    )}
                                    onMouseDown={(event) => {
                                        event.preventDefault();
                                        pick(hit.id);
                                    }}
                                    onMouseEnter={() => {
                                        setActive(position);
                                    }}
                                >
                                    <Icon size={14} className={toneClasses[kindTone(held?.kind ?? "")].ink} />
                                    <span className="flex-1 truncate">{held?.name ?? hit.id}</span>
                                    <span className="text-2xs text-ink-faint">{copy.graph.search.field[hit.field]}</span>
                                </li>
                            );
                        })
                    )}
                </ul>
            ) : null}
        </div>
    );
}

export interface ToolbarProps {
    index: GraphIndex;
    view: GraphView;
    selectedId: string | null;
    connectFrom: string | null;
    reviewing: boolean;
    onView: (view: GraphView) => void;
    onPick: (id: string) => void;
    onCreate: () => void;
    onConnect: () => void;
    onStopConnect: () => void;
    onReview: () => void;
    focusSearch: number;
}

export function Toolbar({
    index,
    view,
    selectedId,
    connectFrom,
    reviewing,
    onView,
    onPick,
    onCreate,
    onConnect,
    onStopConnect,
    onReview,
    focusSearch,
}: ToolbarProps): ReactElement {
    const proposed = index.counts.proposedEdges;
    return (
        <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
            <div className="flex min-w-0 items-center gap-2">
                <Button size="sm" variant="primary" icon={AddIcon} onClick={onCreate}>
                    {copy.graph.inspector.title}
                    <Key>N</Key>
                </Button>
                {connectFrom === null ? (
                    <Button size="sm" variant="secondary" icon={AddLinkIcon} disabled={selectedId === null} onClick={onConnect}>
                        {copy.graph.connect.start}
                        <Key>E</Key>
                    </Button>
                ) : (
                    <Button size="sm" variant="secondary" icon={CloseIcon} className="text-info" onClick={onStopConnect}>
                        {copy.graph.connect.stop}
                    </Button>
                )}
                <span className="h-4 w-px bg-hairline" aria-hidden={true} />
                <SearchBox index={index} onPick={onPick} focusKey={focusSearch} />
            </div>
            <div className="flex shrink-0 items-center gap-2">
                {proposed > 0 || reviewing ? (
                    <Button size="sm" variant={reviewing ? "primary" : "secondary"} icon={PolylineIcon} aria-pressed={reviewing} onClick={onReview}>
                        {copy.graph.queue.open}
                        <span className={cx("ml-1 rounded-sm px-1 font-mono text-2xs", reviewing ? "bg-on-accent/20" : "bg-info-soft text-info")}>{proposed}</span>
                    </Button>
                ) : null}
                <span className="font-mono text-2xs text-ink-faint">{copy.graph.total(index.counts.total)}</span>
                <div className="flex shrink-0 items-center gap-0.5 rounded-md bg-inset p-0.5">
                    <ViewButton view="map" current={view} label={copy.graph.views.map} icon={HubIcon} onSelect={onView} />
                    <ViewButton view="outline" current={view} label={copy.graph.views.outline} icon={TableRowsIcon} onSelect={onView} />
                </div>
            </div>
        </header>
    );
}
