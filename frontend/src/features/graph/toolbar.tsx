import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { entityKinds } from "../../generated/vocab.js";
import { Button, cx, LinkIcon, Segmented, Switch, Toolbar, toneClasses } from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { entityIcon, kindLabel, kindTone, lensHint, lensLabel } from "./labels.js";
import type { GraphIndex } from "./model/index.js";
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
    const toggleKind = (kind: string): void => {
        const held = new Set(query.kinds);
        if (held.has(kind)) {
            held.delete(kind);
        } else {
            held.add(kind);
        }
        onChange({ ...query, kinds: entityKinds.filter((known) => held.has(known)) });
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
            <div className="flex items-center gap-0.5" role="group" aria-label={copy.graph.lens.kinds}>
                {entityKinds.map((kind) => {
                    const Icon = entityIcon(kind);
                    const active = query.kinds.includes(kind);
                    return (
                        <button
                            key={kind}
                            type="button"
                            aria-pressed={active}
                            title={kindLabel(kind)}
                            aria-label={kindLabel(kind)}
                            className={cx(
                                "flex h-6 w-6 items-center justify-center rounded-md transition-colors duration-100",
                                active ? `bg-raised ${toneClasses[kindTone(kind)].ink}` : "text-ink-faint hover:bg-inset hover:text-ink-dim",
                            )}
                            onClick={() => {
                                toggleKind(kind);
                            }}
                        >
                            <Icon size={14} />
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
                disabled={query.lens === "all" && query.kinds.length === 0}
                onChange={(event) => {
                    onChange({ ...query, isolate: event.target.checked });
                }}
            />
        </Toolbar>
    );
}
