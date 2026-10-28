import { useCallback, useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { useRules } from "../../../data/hooks/classification.js";
import { usePagePaths } from "../../../data/hooks/pages.js";
import { usePageTypes } from "../../../data/hooks/templates.js";
import type { PageType, Rule } from "../../../data/types.js";
import { orderedTypes } from "../types/model.js";
import type { Names } from "./model.js";

const said = copy.classification;

export interface RuleNaming {
    listed: ReturnType<typeof useRules>;
    typed: ReturnType<typeof usePageTypes>;
    rules: readonly Rule[];
    fallback: Rule | null;
    beforeRules: number;
    types: readonly PageType[];
    byType: ReadonlyMap<string, PageType>;
    typeName: (typeId: string) => string;
    ruleName: (ruleId: string) => string;
    names: Names;
}

export function useRuleNaming(siteId: string): RuleNaming {
    const listed = useRules(siteId);
    const typed = usePageTypes(siteId);
    const rules = useMemo(() => listed.data?.rules ?? [], [listed.data]);
    const fallback = listed.data?.fallback ?? null;
    const types = useMemo(() => orderedTypes(typed.data?.types ?? []), [typed.data]);
    const byType = useMemo(() => new Map(types.map((type) => [type.id, type] as const)), [types]);
    const underIds = useMemo(
        () => [...new Set(rules.map((rule) => rule.conditions.ancestor?.pageId ?? "").filter((id) => id !== ""))],
        [rules],
    );
    const paths = usePagePaths(underIds, underIds.length > 0);

    const typeName = useCallback((typeId: string): string => byType.get(typeId)?.name ?? said.lostType, [byType]);
    const ruleName = useCallback(
        (ruleId: string): string => {
            const held = rules.find((rule) => rule.id === ruleId);
            if (held === undefined) {
                return ruleId === fallback?.id ? said.fallback.name : said.previewDialog.noRule;
            }
            return said.ruleName(held.number, held.name === "" ? said.unnamed : held.name);
        },
        [rules, fallback],
    );
    const names: Names = useMemo(() => ({ type: typeName, page: (pageId) => paths.get(pageId) ?? said.lostPage }), [typeName, paths]);

    return {
        listed,
        typed,
        rules,
        fallback,
        beforeRules: listed.data?.beforeRules ?? 0,
        types,
        byType,
        typeName,
        ruleName,
        names,
    };
}
