import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useRunParents } from "../../data/hooks/runs.js";
import type { SiteMapNode } from "../../data/types.js";
import { Segmented, useDebounced } from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import type { NamedPage } from "../pages/pick/model.js";
import { PageTree } from "../pages/pick/tree.js";
import { startRefusal } from "./start-model.js";
import type { Scope } from "./start-selection.js";
import { idsOf, inScope, pickable, requiredOf, scopes, selectionKey } from "./start-selection.js";

const parentsDelayMs = 300;

const scopeOptions: readonly SegmentedOption<Scope>[] = scopes.map((scope) => ({
    value: scope,
    label: copy.runs.start.scopes[scope],
}));

export interface TargetTreeProps {
    siteId: string;
    selected: ReadonlySet<string>;
    problem?: string | null;
    onChange: (next: ReadonlySet<string>, named: readonly NamedPage[]) => void;
}

export function TargetTree({ siteId, selected, problem, onChange }: TargetTreeProps): ReactElement {
    const [scope, setScope] = useState<Scope>("all");
    const settled = useDebounced(selectionKey(selected), parentsDelayMs);
    const asked = useMemo(() => idsOf(settled), [settled]);
    const parents = useRunParents(siteId, asked);
    const required = useMemo(() => requiredOf(parents.data?.added, selected), [parents.data, selected]);
    const keep = useMemo(() => inScope(scope), [scope]);
    const refused = startRefusal(asked.length === 0 ? null : parents.error);

    const refusalOf = (node: SiteMapNode): string | null => {
        if (!pickable(node)) {
            return copy.runs.start.unmapped;
        }
        const neededBy = required.get(node.page.pageId);
        return neededBy === undefined ? null : copy.runs.start.required(neededBy);
    };

    return (
        <PageTree
            siteId={siteId}
            selected={selected}
            onChange={onChange}
            pickable={pickable}
            keep={keep}
            required={required}
            label={copy.runs.start.pages}
            picked={copy.runs.start.picked(selected.size, required.size)}
            refusalOf={refusalOf}
            noteOf={(node) => (required.has(node.page.pageId) ? copy.runs.start.writtenFirst : null)}
            filter={
                <Segmented
                    label={copy.runs.start.scope}
                    size="sm"
                    value={scope}
                    options={scopeOptions}
                    onValueChange={setScope}
                />
            }
            problem={problem ?? refused.targets ?? refused.banner}
        />
    );
}
