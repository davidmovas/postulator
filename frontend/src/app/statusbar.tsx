import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useRuns } from "../data/hooks/runs.js";
import { useUsage } from "../data/hooks/models.js";
import { usePluginState } from "../data/hooks/sync.js";
import { useLockGate } from "../data/lock.js";
import { usd } from "../domain/format.js";
import {
    ExtensionIcon,
    ExtensionOffIcon,
    LockIcon,
    LockOpenIcon,
    PlayCircleIcon,
    cx,
} from "../ui/index.js";

export interface StatusBarProps {
    siteId: string | null;
}

export function StatusBar({ siteId }: StatusBarProps) {
    const running = useRuns({ status: "running" }, null, 20);
    const usage = useUsage();
    const plugin = usePluginState(siteId);
    const gate = useLockGate();

    const active = flatten(running.data?.pages).length;
    const spent = usage.data?.usd ?? 0;
    const installed = plugin.data?.plugin.installed === true;

    return (
        <footer className="flex h-6 shrink-0 items-center gap-4 border-t border-hairline bg-panel px-3 text-2xs text-ink-faint">
            <span className="flex items-center gap-1">
                <PlayCircleIcon size={13} className={cx(active > 0 ? "text-info" : undefined)} />
                {copy.shell.activeRuns}: <span className="font-mono text-ink-dim">{active}</span>
            </span>
            <span className="flex items-center gap-1">
                {copy.shell.spend}: <span className="font-mono text-ink-dim">{usd(spent)}</span>
            </span>
            {siteId === null ? null : (
                <span className={cx("flex items-center gap-1", installed ? undefined : "text-warn")}>
                    {installed ? <ExtensionIcon size={13} /> : <ExtensionOffIcon size={13} />}
                    {installed ? copy.shell.pluginInstalled : copy.shell.pluginMissing}
                </span>
            )}
            <span className="ml-auto flex items-center gap-1">
                {gate.locked ? <LockIcon size={13} /> : <LockOpenIcon size={13} />}
                {gate.locked
                    ? copy.shell.locked
                    : gate.protectedByPassword
                      ? copy.shell.unlocked
                      : copy.shell.unprotected}
            </span>
        </footer>
    );
}
