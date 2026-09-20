import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx, SectionLabel } from "../../ui/index.js";
import type { GroupKey } from "./outline.js";
import { groupKeys, groupTitles } from "./outline.js";

export interface OutlineRailProps {
    active: GroupKey;
    changed: ReadonlySet<GroupKey>;
    onSelect: (key: GroupKey) => void;
}

export function OutlineRail({ active, changed, onSelect }: OutlineRailProps): ReactElement {
    return (
        <nav aria-label={copy.templates.editor.groups.title} className="flex flex-col gap-0.5 p-2">
            <SectionLabel className="px-2 pb-1.5">{copy.templates.editor.groups.title}</SectionLabel>
            {groupKeys.map((key) => (
                <button
                    key={key}
                    type="button"
                    data-editor-group={key}
                    aria-current={key === active}
                    onClick={() => {
                        onSelect(key);
                    }}
                    className={cx(
                        "flex h-7 items-center gap-2 rounded-md px-2 text-left text-sm transition-colors duration-100",
                        key === active
                            ? "bg-raised font-medium text-ink"
                            : "text-ink-dim hover:bg-inset hover:text-ink",
                    )}
                >
                    <span className="min-w-0 flex-1 truncate">{groupTitles[key]}</span>
                    {changed.has(key) ? (
                        <span aria-hidden={true} className="h-1.5 w-1.5 shrink-0 rounded-full bg-accent" />
                    ) : null}
                </button>
            ))}
        </nav>
    );
}
