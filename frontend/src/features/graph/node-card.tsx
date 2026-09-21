import type { ReactElement } from "react";

import type { Point } from "../../canvas/viewport.js";
import { copy } from "../../copy/index.js";
import { StatusBadge, toneClasses } from "../../ui/index.js";
import type { Tone } from "../../ui/index.js";
import { entityIcon, formatScore, kindLabel, kindTone } from "./labels.js";
import type { GraphIndex } from "./model/index.js";

const cardWidth = 224;
const offset = 14;

export interface Hovered {
    id: string;
    at: Point;
}

interface Flag {
    key: string;
    label: string;
    tone: Tone;
}

export function flagsOf(index: GraphIndex, id: string): readonly Flag[] {
    const problems = index.problems.get(id);
    const out: Flag[] = [];
    if (problems?.noPage === true) {
        out.push({ key: "noPage", label: copy.graph.card.noPage, tone: "danger" });
    }
    if (index.placementProposed.has(id)) {
        out.push({ key: "placement", label: copy.graph.card.proposedPlacement, tone: "info" });
    }
    if ((problems?.proposed ?? 0) > 0) {
        out.push({ key: "proposed", label: copy.graph.card.proposed(problems?.proposed ?? 0), tone: "info" });
    }
    if (problems?.multiParent === true) {
        out.push({ key: "parents", label: copy.graph.card.extraParent, tone: "muted" });
    }
    if (problems?.orphan === true) {
        out.push({ key: "orphan", label: copy.graph.card.orphan, tone: "warn" });
    }
    return out;
}

export interface NodeCardProps {
    index: GraphIndex;
    hovered: Hovered;
    hostWidth: number;
}

export function NodeCard({ index, hovered, hostWidth }: NodeCardProps): ReactElement | null {
    const entity = index.byId.get(hovered.id);
    if (entity === undefined) {
        return null;
    }
    const Icon = entityIcon(entity.kind);
    const flags = flagsOf(index, hovered.id);
    const left = hovered.at.x + offset + cardWidth > hostWidth ? hovered.at.x - offset - cardWidth : hovered.at.x + offset;
    return (
        <div
            role="tooltip"
            className="pointer-events-none absolute z-20 flex flex-col gap-1.5 rounded-md border border-edge bg-raised p-2"
            style={{ left: `${Math.max(left, 0)}px`, top: `${hovered.at.y + offset}px`, width: `${cardWidth}px` }}
        >
            <div className="flex items-center gap-2">
                <Icon size={14} className={toneClasses[kindTone(entity.kind)].ink} />
                <span className="min-w-0 flex-1 truncate text-xs font-semibold text-ink">{entity.name}</span>
            </div>
            <div className="flex items-center gap-2 text-2xs text-ink-dim">
                <span>{kindLabel(entity.kind)}</span>
                <span className="font-mono">{formatScore(entity.score)}</span>
            </div>
            {flags.length === 0 ? null : (
                <div className="flex flex-wrap gap-1">
                    {flags.map((flag) => (
                        <StatusBadge key={flag.key} tone={flag.tone} dot={false}>
                            {flag.label}
                        </StatusBadge>
                    ))}
                </div>
            )}
        </div>
    );
}
