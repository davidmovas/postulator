import type { ReactElement } from "react";
import { Fragment } from "react";

import { copy } from "../../copy/index.js";
import { ChevronRightIcon, Skeleton } from "../../ui/index.js";
import { useBranch } from "./loader.js";

const said = copy.map;

const crumb = "shrink-0 rounded-sm px-1 text-ink-dim hover:bg-inset hover:text-ink";

export interface SubtreeCrumbsProps {
    siteId: string;
    under: string;
    onUnder: (pageId: string) => void;
}

export function SubtreeCrumbs({ siteId, under, onUnder }: SubtreeCrumbsProps): ReactElement {
    const root = useBranch({ siteId, parentId: under });
    const parent = root.view?.parent?.page ?? null;
    const trail = root.view?.trail ?? [];
    return (
        <nav aria-label={said.subtree} className="flex h-8 shrink-0 items-center gap-1 overflow-hidden border-b border-hairline px-3 text-xs">
            <button
                type="button"
                className={crumb}
                onClick={() => {
                    onUnder("");
                }}
            >
                {said.whole}
            </button>
            {trail.map((ref) => (
                <Fragment key={ref.pageId}>
                    <ChevronRightIcon size={13} className="shrink-0 text-ink-faint" />
                    <button
                        type="button"
                        title={ref.path}
                        className={crumb}
                        onClick={() => {
                            onUnder(ref.pageId);
                        }}
                    >
                        {ref.title === "" ? ref.path : ref.title}
                    </button>
                </Fragment>
            ))}
            <ChevronRightIcon size={13} className="shrink-0 text-ink-faint" />
            {parent === null ? (
                <Skeleton width="10ch" height={10} />
            ) : (
                <span aria-current="page" className="flex min-w-0 items-center gap-1.5 px-1">
                    <span className="truncate font-medium text-ink">{parent.title === "" ? said.untitled : parent.title}</span>
                    <span className="truncate font-mono text-2xs text-ink-faint">{parent.path}</span>
                </span>
            )}
        </nav>
    );
}
