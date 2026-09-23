import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../copy/index.js";
import { cx, SearchIcon, toneClasses } from "../../ui/index.js";
import { entityIcon, kindTone } from "./labels.js";
import type { GraphIndex } from "./model/index.js";
import { rank } from "./model/search.js";

const hitLimit = 8;

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
            <label className="flex h-7 w-44 items-center gap-1 rounded-md border border-hairline bg-inset px-2 focus-within:border-accent">
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
                <ul role="listbox" className="absolute top-8 left-0 z-20 w-72 overflow-hidden rounded-md border border-edge bg-raised py-1">
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
