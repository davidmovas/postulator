import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useTools } from "../../data/hooks/tools.js";
import type { Tool } from "../../data/types.js";
import { Drawer, EmptyState, Input, SearchIcon, SectionLabel, SkeletonRows, StatusBadge, toneClasses } from "../../ui/index.js";
import type { Family } from "./conversation/model/tools.js";
import { familyOf, verbOf } from "./conversation/model/tools.js";
import { familyIcon, familyLabel, riskLabel, riskTone } from "./labels.js";

const familyOrder: readonly Family[] = ["sites", "graph", "pages", "templates", "policies", "runs", "sync", "reports", "imports", "models", "content", "schedules", "other"];

function grouped(tools: readonly Tool[], query: string): [Family, Tool[]][] {
    const needle = query.trim().toLowerCase();
    const buckets = new Map<Family, Tool[]>();
    for (const tool of tools) {
        if (needle !== "" && !tool.name.toLowerCase().includes(needle) && !tool.description.toLowerCase().includes(needle)) {
            continue;
        }
        const family = familyOf(tool.name);
        const held = buckets.get(family) ?? [];
        held.push(tool);
        buckets.set(family, held);
    }
    return familyOrder.filter((family) => buckets.has(family)).map((family) => [family, buckets.get(family) ?? []]);
}

export interface ToolsDrawerProps {
    onClose: () => void;
}

export function ToolsDrawer({ onClose }: ToolsDrawerProps): ReactElement {
    const tools = useTools();
    const [query, setQuery] = useState("");
    const listed = useMemo(() => tools.data?.tools ?? [], [tools.data]);
    const groups = useMemo(() => grouped(listed, query), [listed, query]);

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={copy.agent.tools.title}
            description={copy.agent.tools.subtitle(listed.length)}
            closeLabel={copy.app.dismiss}
            width={560}
        >
            <div className="flex h-full min-h-0 flex-col">
                <div className="flex shrink-0 flex-col gap-2 border-b border-hairline p-3">
                    <div className="relative">
                        <SearchIcon size={14} className="pointer-events-none absolute top-1/2 left-2 -translate-y-1/2 text-ink-faint" />
                        <Input
                            aria-label={copy.agent.tools.search}
                            placeholder={copy.agent.tools.search}
                            value={query}
                            className="pl-7"
                            onChange={(event) => {
                                setQuery(event.target.value);
                            }}
                        />
                    </div>
                    <div className="grid grid-cols-3 gap-2 text-2xs text-ink-dim">
                        {(["read", "write", "dangerous"] as const).map((risk) => (
                            <div key={risk} className="flex flex-col gap-0.5">
                                <StatusBadge tone={riskTone(risk)} className="self-start">
                                    {riskLabel(risk)}
                                </StatusBadge>
                                <span>{copy.agent.tools[`${risk}Body`]}</span>
                            </div>
                        ))}
                    </div>
                </div>
                <div className="min-h-0 flex-1 overflow-auto p-3">
                    {tools.isPending ? (
                        <SkeletonRows rows={10} label={copy.app.loading} />
                    ) : groups.length === 0 ? (
                        <EmptyState icon={SearchIcon} title={copy.agent.tools.noMatch} body={listed.length === 0 ? copy.empty.tools : copy.agent.tools.search} />
                    ) : (
                        <div className="flex flex-col gap-4">
                            {groups.map(([family, held]) => {
                                const Icon = familyIcon(family);
                                return (
                                    <section key={family} className="flex flex-col gap-1.5">
                                        <div className="flex items-center gap-1.5">
                                            <Icon size={14} className="text-ink-faint" />
                                            <SectionLabel>{familyLabel(family)}</SectionLabel>
                                            <span className="font-mono text-2xs text-ink-faint">{held.length}</span>
                                        </div>
                                        <ul className="flex flex-col divide-y divide-hairline rounded-md border border-hairline bg-inset">
                                            {held.map((tool) => (
                                                <li key={tool.name} className="flex flex-col gap-0.5 px-2.5 py-1.5">
                                                    <div className="flex items-center gap-2">
                                                        <span className="text-xs font-semibold text-ink">{verbOf(tool.name)}</span>
                                                        <span className={`text-2xs font-semibold tracking-label uppercase ${toneClasses[riskTone(tool.risk)].ink}`}>
                                                            {riskLabel(tool.risk)}
                                                        </span>
                                                        <span className="ml-auto font-mono text-2xs text-ink-faint">{tool.name}</span>
                                                    </div>
                                                    <p className="text-xs text-ink-dim">{tool.description}</p>
                                                </li>
                                            ))}
                                        </ul>
                                    </section>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>
        </Drawer>
    );
}
