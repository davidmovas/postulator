import * as Popover from "@radix-ui/react-popover";
import type { ReactElement } from "react";
import { useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";
import type { Site } from "../data/types.js";
import { siteStatusTone } from "../features/sites/index.js";
import { AddIcon, CheckIcon, PublicIcon, UnfoldMoreIcon, cx, noDrag, toneClasses } from "../ui/index.js";

function Dot({ site }: { site: Site }): ReactElement {
    return (
        <span
            aria-hidden={true}
            className={cx("h-1.5 w-1.5 shrink-0 rounded-full", toneClasses[siteStatusTone(site.status)].solid)}
        />
    );
}

interface RowProps {
    site: Site;
    current: boolean;
    onPick: (id: string) => void;
}

function Row({ site, current, onPick }: RowProps): ReactElement {
    return (
        <button
            type="button"
            onClick={() => {
                onPick(site.id);
            }}
            className={cx(
                "flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left",
                current ? "bg-inset" : "hover:bg-inset",
            )}
        >
            <Dot site={site} />
            <span className="flex min-w-0 flex-1 flex-col">
                <span className="truncate text-sm font-semibold text-ink">{site.name}</span>
                <span className="truncate font-mono text-2xs text-ink-faint">{site.baseUrl}</span>
            </span>
            {current ? <CheckIcon size={14} className="shrink-0 text-accent" /> : null}
        </button>
    );
}

export interface SitePillProps {
    siteId: string | null;
}

export function SitePill({ siteId }: SitePillProps): ReactElement {
    const navigate = useNavigate();
    const sites = useSites();
    const [open, setOpen] = useState(false);
    const rows = flatten(sites.data?.pages);
    const current = rows.find((site) => site.id === siteId) ?? null;

    const leave = (to: string): void => {
        setOpen(false);
        void navigate(to);
    };

    return (
        <Popover.Root open={open} onOpenChange={setOpen}>
            <Popover.Trigger asChild={true}>
                <button
                    type="button"
                    style={noDrag}
                    aria-label={copy.shell.siteSwitcher}
                    className="flex h-[30px] max-w-70 shrink-0 items-center gap-2 rounded-md border border-hairline bg-transparent px-2.5 text-left transition-colors duration-100 ease-out hover:bg-inset"
                >
                    {current === null ? (
                        <PublicIcon size={14} className="shrink-0 text-ink-faint" />
                    ) : (
                        <Dot site={current} />
                    )}
                    <span className="flex min-w-0 flex-col items-start">
                        <span className="max-w-full truncate text-sm leading-[14px] font-semibold whitespace-nowrap text-ink">
                            {current === null ? copy.shell.chooseSite : current.name}
                        </span>
                        {current === null ? null : (
                            <span className="max-w-full truncate font-mono text-2xs leading-[12px] text-ink-faint">
                                {current.baseUrl}
                            </span>
                        )}
                    </span>
                    <UnfoldMoreIcon size={16} className="shrink-0 text-ink-dim" />
                </button>
            </Popover.Trigger>
            <Popover.Portal>
                <Popover.Content
                    align="start"
                    sideOffset={4}
                    className="z-50 w-70 rounded-lg border border-edge bg-panel p-1 data-[state=open]:animate-fade-in"
                >
                    {rows.map((site) => (
                        <Row
                            key={site.id}
                            site={site}
                            current={site.id === siteId}
                            onPick={(id) => {
                                leave(`/s/${id}/overview`);
                            }}
                        />
                    ))}
                    {rows.length === 0 ? null : <div aria-hidden={true} className="my-1 h-px bg-hairline" />}
                    <button
                        type="button"
                        onClick={() => {
                            leave("/sites");
                        }}
                        className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-xs text-ink-soft hover:bg-inset hover:text-ink"
                    >
                        <PublicIcon size={14} className="shrink-0" />
                        {copy.shell.allSites}
                    </button>
                    <button
                        type="button"
                        onClick={() => {
                            leave("/sites?action=new");
                        }}
                        className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-xs text-ink-soft hover:bg-inset hover:text-ink"
                    >
                        <AddIcon size={14} className="shrink-0 text-accent" />
                        {copy.sites.add}
                    </button>
                </Popover.Content>
            </Popover.Portal>
        </Popover.Root>
    );
}
