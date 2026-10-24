import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { isBrowsable, openExternal } from "../../../data/host.js";
import type { ModelLink } from "../../../data/types.js";
import { BoltIcon, OpenInNewIcon, Panel, PanelHeader, PublicIcon } from "../../../ui/index.js";
import { Note, PageRef } from "./facts.js";
import { linkSourceLabel, linkTypeLabel, regionKindLabel } from "./labels.js";

const said = copy.pages.model.links;

const contextual = "contextual";

function Target({ link, siteId, search }: { link: ModelLink; siteId: string; search: string }): ReactElement {
    if (link.toPageId !== "") {
        return <PageRef pageId={link.toPageId} siteId={siteId} search={search} label={link.toPath} />;
    }
    if (isBrowsable(link.toUrl)) {
        return (
            <button
                type="button"
                title={copy.app.openExternal}
                onClick={() => {
                    void openExternal(link.toUrl);
                }}
                className="flex min-w-0 items-center gap-1 text-left text-accent hover:underline"
            >
                <OpenInNewIcon size={12} className="shrink-0" />
                <span className="truncate font-mono text-xs">{link.toUrl}</span>
            </button>
        );
    }
    return (
        <span className="flex min-w-0 items-center gap-1">
            <PublicIcon size={12} className="shrink-0 text-ink-faint" />
            <span className="truncate font-mono text-xs text-ink-soft">{link.toPath === "" ? link.toUrl : link.toPath}</span>
        </span>
    );
}

export interface ModelLinksPaneProps {
    links: readonly ModelLink[];
    siteId: string;
    search: string;
}

export function ModelLinksPane({ links, siteId, search }: ModelLinksPaneProps): ReactElement {
    const counted = links.filter((link) => link.type === contextual).length;

    return (
        <Panel>
            <PanelHeader title={said.title}>
                {links.length === 0 ? null : (
                    <span className="shrink-0 font-mono text-2xs text-ink-faint">{said.count(links.length, counted)}</span>
                )}
            </PanelHeader>
            {links.length === 0 ? (
                <Note>{said.none}</Note>
            ) : (
                <ul className="flex flex-col">
                    {links.map((link, index) => (
                        <li
                            key={`${index}:${link.toUrl}`}
                            className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                        >
                            <span className="flex w-28 shrink-0 flex-col">
                                <span className="truncate text-xs text-ink">{linkTypeLabel(link.type)}</span>
                                <span className="truncate text-2xs text-ink-faint">
                                    {link.region === ""
                                        ? linkSourceLabel(link.source)
                                        : copy.pages.linkKind(regionKindLabel(link.region), linkSourceLabel(link.source))}
                                </span>
                            </span>
                            <span className="min-w-0 flex-1">
                                <Target link={link} siteId={siteId} search={search} />
                            </span>
                            <span className="w-1/3 shrink-0 truncate text-right text-xs text-ink-dim">{link.anchor}</span>
                            <span className="flex w-4 shrink-0 justify-end">
                                {link.ours ? (
                                    <span title={said.ours} aria-label={said.ours} className="flex">
                                        <BoltIcon size={13} className="text-accent" />
                                    </span>
                                ) : null}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
        </Panel>
    );
}
