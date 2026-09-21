import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { useUsage } from "../../../data/hooks/models.js";
import { tokens, usd } from "../../../domain/format.js";
import { Panel, PanelHeader, Skeleton } from "../../../ui/index.js";

const said = copy.settings.models.spend;

export function SpendTile(): ReactElement {
    const usage = useUsage();

    return (
        <Panel>
            <PanelHeader title={said.title} />
            <div className="flex flex-col gap-1 p-3">
                {usage.data === undefined ? (
                    <Skeleton height={52} />
                ) : (
                    <>
                        <span className="text-2xl font-semibold tracking-tight text-ink">{usd(usage.data.usd)}</span>
                        <span className="font-mono text-xs text-ink-dim">{said.calls(usage.data.calls)}</span>
                        <span className="font-mono text-xs text-ink-dim">
                            {said.tokens(tokens(usage.data.usage.total))}
                        </span>
                    </>
                )}
            </div>
        </Panel>
    );
}
