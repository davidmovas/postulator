import type { ReactElement, ReactNode } from "react";
import { Link } from "react-router";

import { cx, LinkIcon } from "../../../ui/index.js";

export function Facts({ children }: { children: ReactNode }): ReactElement {
    return <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 px-3 py-2 text-xs">{children}</dl>;
}

export interface FactProps {
    label: string;
    children: ReactNode;
    mono?: boolean;
    title?: string;
}

export function Fact({ label, children, mono = false, title }: FactProps): ReactElement {
    return (
        <>
            <dt className="text-ink-faint">{label}</dt>
            <dd className={cx("min-w-0 truncate text-ink-soft", mono && "font-mono")} title={title}>
                {children}
            </dd>
        </>
    );
}

export function Note({ children }: { children: ReactNode }): ReactElement {
    return <p className="p-3 text-xs text-ink-dim">{children}</p>;
}

export interface PageRefProps {
    pageId: string;
    siteId: string;
    search: string;
    label: string;
    path?: string;
}

export function PageRef({ pageId, siteId, search, label, path }: PageRefProps): ReactElement {
    return (
        <Link to={`/s/${siteId}/pages/${pageId}${search}`} className="flex min-w-0 items-center gap-1">
            <LinkIcon size={12} className="shrink-0" />
            <span className="truncate text-xs">{label}</span>
            {path === undefined || path === label ? null : (
                <span className="shrink-0 font-mono text-2xs text-ink-faint">{path}</span>
            )}
        </Link>
    );
}
