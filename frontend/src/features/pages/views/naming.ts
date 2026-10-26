import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { useRules } from "../../../data/hooks/classification.js";
import { usePageTypes } from "../../../data/hooks/templates.js";
import type { PageType } from "../../../data/types.js";
import { orderedTypes } from "../../settings/types/model.js";
import type { RuleRef, TypeIdOf } from "./model.js";

export interface TypeNaming {
    types: readonly PageType[];
    loaded: boolean;
    typeOf: (typeId: string | null) => PageType | undefined;
    typeName: (typeId: string | null) => string;
    typeIdOf: TypeIdOf;
    ruleOf: (ruleId: string) => RuleRef | undefined;
    ruleName: (ruleId: string) => string;
}

export function useTypeNaming(siteId: string): TypeNaming {
    const listed = usePageTypes(siteId);
    const rules = useRules(siteId);
    return useMemo(() => {
        const types = orderedTypes(listed.data?.types ?? []);
        const byId = new Map(types.map((type) => [type.id, type] as const));
        const byName = new Map(types.map((type) => [type.name, type.id] as const));
        const byRule = new Map((rules.data?.rules ?? []).map((rule) => [rule.id, { number: rule.number, name: rule.name }] as const));
        return {
            types,
            loaded: listed.data !== undefined,
            typeOf: (typeId) => (typeId === null ? undefined : byId.get(typeId)),
            typeName: (typeId) => (typeId === null || typeId === "" ? copy.pageTypes.untyped : (byId.get(typeId)?.name ?? copy.classification.lostType)),
            typeIdOf: (name) => byName.get(name) ?? null,
            ruleOf: (ruleId) => byRule.get(ruleId),
            ruleName: (ruleId) => {
                const rule = byRule.get(ruleId);
                return rule === undefined ? copy.pages.typeReason.ruleGone : copy.pages.typeReason.rule(rule.number, rule.name);
            },
        };
    }, [listed.data, rules.data]);
}
