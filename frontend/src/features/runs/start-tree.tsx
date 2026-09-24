import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { usePageTree } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { Segmented } from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { PageTree } from "../pages/pick/tree.js";
import type { Scope } from "./start-selection.js";
import { inScope, indexTree, pickable, requiredParents, scopes } from "./start-selection.js";

const scopeOptions: readonly SegmentedOption<Scope>[] = scopes.map((scope) => ({
    value: scope,
    label: copy.runs.start.scopes[scope],
}));

export interface TargetTreeProps {
    siteId: string;
    selected: ReadonlySet<string>;
    problem?: string | null;
    onChange: (next: ReadonlySet<string>) => void;
}

export function TargetTree({ siteId, selected, problem, onChange }: TargetTreeProps): ReactElement {
    const tree = usePageTree(siteId);
    const [scope, setScope] = useState<Scope>("all");
    const index = useMemo(() => indexTree(tree.data?.roots ?? null), [tree.data]);
    const required = useMemo(() => requiredParents(selected, index), [selected, index]);
    const keep = useMemo(() => inScope(scope), [scope]);

    const refusalOf = (page: Page): string | null => {
        if (!pickable(page)) {
            return copy.runs.start.unmapped;
        }
        const neededBy = required.get(page.id);
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
            noteOf={(page) => (required.has(page.id) ? copy.runs.start.writtenFirst : null)}
            filter={
                <Segmented
                    label={copy.runs.start.scope}
                    size="sm"
                    value={scope}
                    options={scopeOptions}
                    onValueChange={setScope}
                />
            }
            problem={problem}
        />
    );
}
