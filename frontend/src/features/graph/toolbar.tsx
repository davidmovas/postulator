import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { Button, cx, LinkIcon, Segmented, Switch, Toolbar, toneClasses } from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import type { PresetRole } from "../settings/types/labels.js";
import { presetIcon, presetLabel, presetRoles, presetTone } from "../settings/types/labels.js";
import { lensHint, lensLabel, stateHint, stateLabel, stateTone } from "./labels.js";
import { nodeStates } from "./model/index.js";
import type { GraphIndex, NodeState } from "./model/index.js";
import type { Lens } from "./model/lens.js";
import { lenses } from "./model/lens.js";
import type { GraphQuery } from "./model/params.js";
import { SearchBox } from "./search-box.js";

const lensOptions: readonly SegmentedOption<Lens>[] = lenses.map((lens) => ({
    value: lens,
    label: lensLabel(lens),
    title: lensHint(lens),
}));

export interface GraphToolbarProps {
    index: GraphIndex;
    query: GraphQuery;
    proofBusy: boolean;
    focusSearch: number;
    onChange: (query: GraphQuery) => void;
    onPick: (id: string) => void;
}

export function GraphToolbar({ index, query, proofBusy, focusSearch, onChange, onPick }: GraphToolbarProps): ReactElement {
    const togglePreset = (preset: PresetRole): void => {
        const held = new Set<PresetRole>(query.presets);
        if (held.has(preset)) {
            held.delete(preset);
        } else {
            held.add(preset);
        }
        onChange({ ...query, presets: presetRoles.filter((known) => held.has(known)) });
    };

    const toggleState = (state: NodeState): void => {
        const held = new Set(query.states);
        if (held.has(state)) {
            held.delete(state);
        } else {
            held.add(state);
        }
        onChange({ ...query, states: nodeStates.filter((known) => held.has(known)) });
    };

    return (
        <Toolbar label={copy.graph.label}>
            <SearchBox index={index} onPick={onPick} focusKey={focusSearch} />
            <Segmented
                label={copy.graph.lens.all}
                value={query.lens}
                options={lensOptions}
                onValueChange={(lens) => {
                    onChange({ ...query, lens });
                }}
            />
            <div className="flex items-center gap-0.5" role="group" aria-label={copy.graph.lens.types}>
                {presetRoles.map((preset) => {
                    const Icon = presetIcon(preset);
                    const active = query.presets.includes(preset);
                    return (
                        <button
                            key={preset}
                            type="button"
                            aria-pressed={active}
                            title={presetLabel(preset)}
                            aria-label={presetLabel(preset)}
                            data-graph-preset={preset}
                            className={cx(
                                "flex h-6 w-6 items-center justify-center rounded-md transition-colors duration-100",
                                active ? `bg-raised ${toneClasses[presetTone(preset)].ink}` : "text-ink-faint hover:bg-inset hover:text-ink-dim",
                            )}
                            onClick={() => {
                                togglePreset(preset);
                            }}
                        >
                            <Icon size={14} />
                        </button>
                    );
                })}
            </div>
            <div className="flex items-center gap-0.5" role="group" aria-label={copy.graph.legend.states}>
                {nodeStates.map((state) => {
                    const active = query.states.includes(state);
                    const count = index.counts.states[state];
                    return (
                        <button
                            key={state}
                            type="button"
                            aria-pressed={active}
                            title={`${stateHint(state)} (${String(count)})`}
                            aria-label={stateLabel(state)}
                            className={cx(
                                "flex h-6 items-center gap-1 rounded-md px-1.5 transition-colors duration-100",
                                active ? "bg-raised" : "hover:bg-inset",
                            )}
                            onClick={() => {
                                toggleState(state);
                            }}
                        >
                            <span
                                aria-hidden={true}
                                className={cx("h-2 w-2 rounded-full", toneClasses[stateTone(state)].solid)}
                            />
                            <span className={cx("font-mono text-2xs", active ? "text-ink" : "text-ink-faint")}>{count}</span>
                        </button>
                    );
                })}
            </div>
            <span className="flex-1" />
            <Button
                size="sm"
                variant={query.proof ? "primary" : "ghost"}
                icon={LinkIcon}
                aria-pressed={query.proof}
                title={copy.graph.lens.proofHint}
                busy={proofBusy}
                onClick={() => {
                    onChange({ ...query, proof: !query.proof });
                }}
            >
                {copy.graph.lens.proof}
            </Button>
            <Switch
                label={copy.graph.lens.isolate}
                title={copy.graph.lens.isolateHint}
                checked={query.isolate}
                disabled={query.lens === "all" && query.presets.length === 0}
                onChange={(event) => {
                    onChange({ ...query, isolate: event.target.checked });
                }}
            />
        </Toolbar>
    );
}
