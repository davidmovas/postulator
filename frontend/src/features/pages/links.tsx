import type { ReactElement } from "react";
import { Link } from "react-router";

import { isBrowsable, openExternal } from "../../data/host.js";
import type { PageLink } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { LinkIcon, OpenInNewIcon, Panel, PanelHeader, PublicIcon, toneClasses } from "../../ui/index.js";
import { originIcon, originTone } from "./labels.js";

export interface PageLinksProps {
    links: readonly PageLink[];
    siteId: string;
    search: string;
}

export function PageLinks({ links, siteId, search }: PageLinksProps): ReactElement {
    const generated = links.filter((held) => held.origin === "generated").length;

    return (
        <Panel>
            <PanelHeader title={copy.pages.detail.links}>
                <span className="shrink-0 font-mono text-2xs text-ink-faint">
                    {copy.pages.detail.linksSummary(links.length, generated)}
                </span>
            </PanelHeader>
            {links.length === 0 ? (
                <p className="p-3 text-xs text-ink-dim">{copy.empty.pageLinks}</p>
            ) : (
                <ul className="flex flex-col">
                    {links.map((held) => {
                        const Icon = originIcon(held.origin);
                        const tone = toneClasses[originTone(held.origin)];
                        return (
                            <li
                                key={held.id}
                                className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                            >
                                <span className="flex shrink-0" title={held.origin}>
                                    <Icon size={13} className={tone.ink} />
                                </span>
                                {held.toPageId === null ? (
                                    isBrowsable(held.toUrl) ? (
                                        <button
                                            type="button"
                                            title={copy.app.openExternal}
                                            onClick={() => {
                                                void openExternal(held.toUrl);
                                            }}
                                            className="flex min-w-0 flex-1 items-center gap-1 text-left text-accent hover:underline"
                                        >
                                            <OpenInNewIcon size={12} className="shrink-0" />
                                            <span className="truncate font-mono text-xs">{held.toUrl}</span>
                                        </button>
                                    ) : (
                                        <span className="flex min-w-0 flex-1 items-center gap-1">
                                            <PublicIcon size={12} className="shrink-0 text-ink-faint" />
                                            <span className="truncate font-mono text-xs text-ink-soft">
                                                {held.toUrl}
                                            </span>
                                        </span>
                                    )
                                ) : (
                                    <Link
                                        to={`/s/${siteId}/pages/${held.toPageId}${search}`}
                                        className="flex min-w-0 flex-1 items-center gap-1"
                                    >
                                        <LinkIcon size={12} className="shrink-0" />
                                        <span className="truncate font-mono text-xs">{held.toUrl}</span>
                                    </Link>
                                )}
                                <span className="w-2/5 shrink-0 truncate text-right text-xs text-ink-dim">
                                    {held.anchorText}
                                </span>
                                <span
                                    className="shrink-0 font-mono text-2xs text-ink-faint"
                                    title={absoluteTime(held.observedAt)}
                                >
                                    {relativeTime(held.observedAt)}
                                </span>
                            </li>
                        );
                    })}
                </ul>
            )}
        </Panel>
    );
}
