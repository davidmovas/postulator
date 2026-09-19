import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { entityKinds } from "../../generated/vocab.js";
import { Button, cx, LinkIcon, Switch, toneClasses } from "../../ui/index.js";
import { entityIcon, kindTone, lensHint, lensLabel, lensTone } from "./labels.js";
import type { Lens } from "./model/lens.js";
import { lenses } from "./model/lens.js";
import type { GraphQuery } from "./model/params.js";

export interface LensBarProps {
    query: GraphQuery;
    counts: Record<Lens, number>;
    proofBusy: boolean;
    onChange: (query: GraphQuery) => void;
}

export function LensBar({ query, counts, proofBusy, onChange }: LensBarProps): ReactElement {
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
        <div className="flex h-8 shrink-0 items-center gap-3 border-b border-hairline px-3">
            <div className="flex items-center gap-1" role="group" aria-label={copy.graph.lens.all}>
                {lenses.map((lens) => {
                    const active = query.lens === lens;
                    return (
                        <button
                            key={lens}
                            type="button"
                            aria-pressed={active}
                            title={lensHint(lens)}
                            className={cx(
                                "inline-flex h-6 items-center gap-1.5 rounded-md px-2 text-xs transition-colors duration-100",
                                active ? "bg-raised text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
                            )}
                            onClick={() => {
                                onChange({ ...query, lens });
                            }}
                        >
                            {lens === "all" ? null : (
                                <span className={cx("h-1.5 w-1.5 rounded-full", toneClasses[lensTone(lens)].solid)} aria-hidden={true} />
                            )}
                            {lensLabel(lens)}
                            <span className="font-mono text-2xs text-ink-faint">{counts[lens]}</span>
                        </button>
                    );
                })}
            </div>
            <span className="h-4 w-px bg-hairline" aria-hidden={true} />
            <div className="flex items-center gap-0.5" role="group" aria-label={copy.graph.lens.kinds}>
                {entityKinds.map((kind) => {
                    const Icon = entityIcon(kind);
                    const active = query.kinds.includes(kind);
                    return (
                        <button
                            key={kind}
                            type="button"
                            aria-pressed={active}
                            title={kind}
                            aria-label={kind}
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
        </div>
    );
}
