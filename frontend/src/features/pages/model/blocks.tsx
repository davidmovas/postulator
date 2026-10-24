import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { ModelBlock } from "../../../data/types.js";
import { Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { Note } from "./facts.js";
import { blockKindLabel, bodyKindLabel } from "./labels.js";

const said = copy.pages.model.blocks;

export interface BlocksPaneProps {
    bodyKind: string;
    blocks: readonly ModelBlock[];
}

export function BlocksPane({ bodyKind, blocks }: BlocksPaneProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={said.title}>
                {bodyKind === "" ? null : (
                    <span className="shrink-0 text-2xs text-ink-faint">{bodyKindLabel(bodyKind)}</span>
                )}
            </PanelHeader>
            {bodyKind === "" ? (
                <Note>{said.none}</Note>
            ) : blocks.length === 0 ? (
                <Note>{said.empty}</Note>
            ) : (
                <ol className="flex flex-col">
                    {blocks.map((held) => (
                        <li
                            key={held.index}
                            className="flex min-w-0 items-start gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                        >
                            <span className="w-5 shrink-0 pt-px text-right font-mono text-2xs text-ink-faint">{held.index}</span>
                            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                                <div className="flex min-w-0 items-center gap-2">
                                    <span className="shrink-0 text-xs text-ink">{blockKindLabel(held.kind)}</span>
                                    {held.name === "" ? null : (
                                        <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">{held.name}</span>
                                    )}
                                    <span className="ml-auto flex shrink-0 items-center gap-1">
                                        {held.dynamic ? (
                                            <StatusBadge tone="info" dot={false}>
                                                {said.dynamic}
                                            </StatusBadge>
                                        ) : null}
                                        {held.ours ? (
                                            <StatusBadge tone="accent" dot={false}>
                                                {said.ours}
                                            </StatusBadge>
                                        ) : null}
                                    </span>
                                </div>
                                {held.excerpt === "" ? null : (
                                    <p className="line-clamp-2 text-2xs text-ink-dim">{held.excerpt}</p>
                                )}
                            </div>
                        </li>
                    ))}
                </ol>
            )}
        </Panel>
    );
}
