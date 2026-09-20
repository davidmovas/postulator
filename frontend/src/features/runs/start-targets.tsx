import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { usePages } from "../../data/hooks/pages.js";
import type { PageFilter } from "../../data/types.js";
import { Button, Checkbox, Input, SectionLabel, SkeletonRows } from "../../ui/index.js";

const pageSize = 100;

export interface StartTargetsProps {
    siteId: string;
    selected: readonly string[];
    onToggle: (pageId: string) => void;
}

export function StartTargets({ siteId, selected, onToggle }: StartTargetsProps): ReactElement {
    const [prefix, setPrefix] = useState("");
    const [applied, setApplied] = useState("");

    const filter = useMemo<PageFilter>(
        () => (applied === "" ? { siteId } : { siteId, pathPrefix: applied }),
        [siteId, applied],
    );
    const listed = usePages(filter, { field: "path", desc: false }, pageSize);
    const pages = useMemo(() => flatten(listed.data?.pages), [listed.data]);

    return (
        <div className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.runs.start.pages}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">
                    {copy.runs.start.selected(selected.length)}
                </span>
            </div>
            <Input
                type="search"
                mono={true}
                value={prefix}
                placeholder="/"
                aria-label={copy.runs.start.search}
                onChange={(event) => {
                    setPrefix(event.target.value);
                }}
                onBlur={() => {
                    setApplied(prefix.trim());
                }}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();
                        setApplied(prefix.trim());
                    }
                }}
            />
            <div className="flex max-h-56 min-h-32 flex-col gap-0.5 overflow-auto rounded-md border border-hairline bg-inset p-1.5">
                {listed.isPending ? (
                    <SkeletonRows rows={5} label={copy.pages.loading} />
                ) : pages.length === 0 ? (
                    <p className="p-1 text-xs text-ink-dim">{copy.empty.pages}</p>
                ) : (
                    pages.map((page) => (
                        <div key={page.id} data-run-target={page.id}>
                            <Checkbox
                                checked={selected.includes(page.id)}
                                label={page.path}
                                onChange={() => {
                                    onToggle(page.id);
                                }}
                            />
                        </div>
                    ))
                )}
                {listed.hasNextPage ? (
                    <Button
                        size="sm"
                        variant="ghost"
                        busy={listed.isFetchingNextPage}
                        onClick={() => {
                            void listed.fetchNextPage();
                        }}
                    >
                        {copy.app.loadMore}
                    </Button>
                ) : null}
            </div>
        </div>
    );
}
