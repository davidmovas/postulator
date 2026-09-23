import type { ReactElement } from "react";

import type { TemplateOverride } from "../../../data/types.js";
import type { GroupKey } from "../outline.js";
import type { Layer } from "../patch.js";
import type { SpecDraft } from "../spec.js";
import { ModelsGroup } from "./models.js";
import { OverridesGroup } from "./overrides.js";
import { PoliciesGroup } from "./policies.js";
import { RecipeGroup } from "./recipe.js";
import { RulesGroup } from "./rules.js";
import { SectionsGroup } from "./sections.js";

export interface GroupViewProps {
    group: GroupKey;
    layer: Layer;
    siteId: string;
    draft: SpecDraft;
    below: SpecDraft;
    siteOverride: TemplateOverride | null;
    pageOverrides: readonly TemplateOverride[];
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function GroupView({
    group,
    layer,
    siteId,
    draft,
    below,
    siteOverride,
    pageOverrides,
    error,
    onChange,
}: GroupViewProps): ReactElement {
    switch (group) {
        case "sections":
            return <SectionsGroup draft={draft} below={below} error={error} onChange={onChange} />;
        case "rules":
            return <RulesGroup draft={draft} below={below} error={error} onChange={onChange} />;
        case "models":
            return <ModelsGroup draft={draft} below={below} error={error} onChange={onChange} />;
        case "recipe":
            return <RecipeGroup draft={draft} below={below} error={error} onChange={onChange} />;
        case "overrides":
            return (
                <OverridesGroup
                    layer={layer}
                    below={below}
                    draft={draft}
                    siteOverride={siteOverride}
                    pageOverrides={pageOverrides}
                    onChange={onChange}
                />
            );
        default:
            return <PoliciesGroup siteId={siteId} />;
    }
}
