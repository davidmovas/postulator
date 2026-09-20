import type { ReactElement, ReactNode } from "react";
import { Link } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { usePendingActions } from "../data/hooks/agent.js";
import { useUsage } from "../data/hooks/models.js";
import { useRuns } from "../data/hooks/runs.js";
import { usePluginState } from "../data/hooks/sync.js";
import { useLockGate } from "../data/lock.js";
import { usd } from "../domain/format.js";
import {
    ExtensionIcon,
    ExtensionOffIcon,
    LockIcon,
    LockOpenIcon,
    PendingActionsIcon,
    PlayCircleIcon,
    cx,
} from "../ui/index.js";

interface ItemProps {
    to: string;
    tone?: string;
    children: ReactNode;
}

function Item({ to, tone, children }: ItemProps): ReactElement {
    return (
        <Link
            to={to}
            className={cx("flex items-center gap-1.5 hover:text-ink", tone ?? "text-ink-faint")}
        >
            {children}
        </Link>
    );
}

export interface StatusBarProps {
    siteId: string | null;
}

export function StatusBar({ siteId }: StatusBarProps): ReactElement {
    const running = useRuns(siteId === null ? { status: "running" } : { siteId, status: "running" }, null, 20);
    const usage = useUsage();
    const plugin = usePluginState(siteId);
    const gate = useLockGate();
    const pending = usePendingActions({ status: "pending" }, 100);

    const active = flatten(running.data?.pages).length;
    const awaiting = flatten(pending.data?.pages).length;
    const spent = usage.data?.usd ?? 0;
    const installed = plugin.data?.plugin.installed === true;
    const version = plugin.data?.plugin.version ?? "";

    return (
        <footer className="flex h-6 shrink-0 items-center gap-4 border-t border-hairline bg-panel px-3 font-mono text-2xs text-ink-faint">
            {siteId === null ? null : (
                <Item to={`/s/${siteId}/runs`} tone={active > 0 ? "text-info" : undefined}>
                    <PlayCircleIcon size={13} />
                    {copy.shell.activeRuns}: {active}
                </Item>
            )}
            <Item to="/settings/models">
                {copy.shell.spend}: {usd(spent)}
            </Item>
            {siteId === null ? null : (
                <Item to="/sites" tone={installed ? undefined : "text-warn"}>
                    {installed ? <ExtensionIcon size={13} /> : <ExtensionOffIcon size={13} />}
                    {installed && version !== ""
                        ? `${copy.shell.pluginInstalled} ${version}`
                        : installed
                          ? copy.shell.pluginInstalled
                          : copy.shell.pluginMissing}
                </Item>
            )}
            {awaiting === 0 ? null : (
                <Item to="/agent/inbox" tone="text-warn">
                    <PendingActionsIcon size={13} />
                    {copy.agent.screen.awaiting(awaiting)}
                </Item>
            )}
            <span className="ml-auto flex items-center">
                <Item to="/settings/security">
                    {gate.locked ? <LockIcon size={13} /> : <LockOpenIcon size={13} />}
                    {gate.locked
                        ? copy.shell.locked
                        : gate.protectedByPassword
                          ? copy.shell.unlocked
                          : copy.shell.unprotected}
                </Item>
            </span>
        </footer>
    );
}
