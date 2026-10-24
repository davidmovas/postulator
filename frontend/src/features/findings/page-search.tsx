import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useSiteMap } from "../../data/hooks/pages.js";
import type { SiteMapRef } from "../../data/types.js";
import { Button, cx, Input, SkeletonRows, useDebounced } from "../../ui/index.js";

const said = copy.findings;

const searchDelayMs = 250;

export interface PageSearchProps {
    siteId: string;
    picked: string | null;
    onPick: (page: SiteMapRef) => void;
}

export function PageSearch({ siteId, picked, onPick }: PageSearchProps): ReactElement {
    const [typed, setTyped] = useState("");
    const search = useDebounced(typed.trim(), searchDelayMs);
    const found = useSiteMap({ siteId, search }, search !== "");
    const pages = useMemo(() => (found.data?.pages ?? []).flatMap((view) => view.nodes.items ?? []).map((node) => node.page), [found.data]);

    const listed = ((): ReactElement => {
        if (search === "") {
            return <p className="p-1 text-xs text-ink-dim">{said.search.typeMore}</p>;
        }
        if (found.isPending) {
            return <SkeletonRows rows={4} label={said.search.label} />;
        }
        if (pages.length === 0) {
            return <p className="p-1 text-xs text-ink-dim">{said.search.none}</p>;
        }
        return (
            <ul role="listbox" aria-label={said.search.label} className="flex flex-col">
                {pages.map((page) => (
                    <li key={page.pageId}>
                        <button
                            type="button"
                            role="option"
                            aria-selected={picked === page.pageId}
                            onClick={() => {
                                onPick(page);
                            }}
                            className={cx(
                                "flex h-7 w-full min-w-0 items-center gap-2 rounded-sm px-1.5 text-left text-xs",
                                picked === page.pageId ? "bg-accent-soft text-ink" : "text-ink-soft hover:bg-raised hover:text-ink",
                            )}
                        >
                            <span className="min-w-0 truncate">{page.title}</span>
                            <span className="ml-auto shrink-0 truncate font-mono text-2xs text-ink-faint">{page.path}</span>
                        </button>
                    </li>
                ))}
            </ul>
        );
    })();

    return (
        <div className="flex flex-col gap-1.5">
            <Input
                type="search"
                value={typed}
                placeholder={said.search.hint}
                aria-label={said.search.label}
                onChange={(event) => {
                    setTyped(event.target.value);
                }}
            />
            <div className="flex max-h-56 min-h-24 flex-col overflow-auto rounded-md border border-hairline bg-inset p-1">
                {listed}
                {found.hasNextPage ? (
                    <Button
                        size="sm"
                        variant="ghost"
                        busy={found.isFetchingNextPage}
                        onClick={() => {
                            void found.fetchNextPage();
                        }}
                    >
                        {said.more}
                    </Button>
                ) : null}
            </div>
        </div>
    );
}
