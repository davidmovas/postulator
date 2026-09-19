import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useRuns } from "../data/hooks/runs.js";
import { useUsage } from "../data/hooks/models.js";
import { usePluginState } from "../data/hooks/sync.js";
import { useLockGate } from "../data/lock.js";
import { usd } from "../domain/format.js";

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

    return (
        <footer className="flex h-6 shrink-0 items-center gap-4 border-t border-base-700 bg-base-900 px-3 text-2xs text-ink-400">
            <span>
                {copy.shell.activeRuns}: {active}
            </span>
            <span>
                {copy.shell.spend}: {usd(spent)}
            </span>
            {siteId !== null && (
                <span>{plugin.data?.plugin.installed === true ? copy.shell.pluginInstalled : copy.shell.pluginMissing}</span>
            )}
            <span className="ml-auto">
                {gate.locked
                    ? copy.shell.locked
                    : gate.protectedByPassword
                      ? copy.shell.unlocked
                      : copy.shell.unprotected}
            </span>
        </footer>
    );
}
