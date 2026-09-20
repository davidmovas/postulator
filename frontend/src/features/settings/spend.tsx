import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useUsage } from "../../data/hooks/models.js";
import { tokens, usd } from "../../domain/format.js";
import { Panel, PanelHeader, Skeleton } from "../../ui/index.js";

function Figure({ label, value }: { label: string; value: string }): ReactElement {
    return (
        <div className="flex flex-col gap-0.5">
            <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</span>
            <span className="font-mono text-sm text-ink">{value}</span>
        </div>
    );
}

export function SpendPanel(): ReactElement {
    const usage = useUsage();

    return (
        <Panel>
            <PanelHeader title={copy.settings.models.spend.title} />
            <p className="border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                {copy.settings.models.spend.blurb}
            </p>
            <div className="p-3">
                {usage.data === undefined ? (
                    <Skeleton height={48} />
                ) : (
                    <div className="grid grid-cols-5 gap-4">
                        <Figure label={copy.settings.models.spend.spent} value={usd(usage.data.usd)} />
                        <Figure label={copy.settings.models.spend.input} value={tokens(usage.data.usage.input)} />
                        <Figure label={copy.settings.models.spend.output} value={tokens(usage.data.usage.output)} />
                        <Figure label={copy.settings.models.spend.total} value={tokens(usage.data.usage.total)} />
                        <Figure label={copy.settings.models.spend.calls} value={String(usage.data.calls)} />
                    </div>
                )}
            </div>
        </Panel>
    );
}
