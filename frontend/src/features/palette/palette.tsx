import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../copy/index.js";
import { SearchIcon, useDebounced } from "../../ui/index.js";
import { PaletteFooter, PaletteList } from "./list.js";
import { usePaletteSections } from "./sections.js";
import type { Destination } from "./sections.js";
import { closePalette, usePaletteOpen } from "./state.js";

const debounceMs = 180;

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
    const [query, setQuery] = useState("");
    const [active, setActive] = useState(0);
    const settled = useDebounced(query.trim(), debounceMs);
    const listRef = useRef<HTMLDivElement | null>(null);
    const sections = usePaletteSections(siteId, destinations, settled);

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
                    <PaletteList listRef={listRef} siteId={siteId} sections={sections} rows={rows} at={at} onActive={setActive} />
                    <PaletteFooter />
                </RadixDialog.Content>
            </RadixDialog.Portal>
        </RadixDialog.Root>
    );
}
