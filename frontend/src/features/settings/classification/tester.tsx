import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import { useRuleTest } from "../../../data/hooks/classification.js";
import { usePagePaths } from "../../../data/hooks/pages.js";
import type { PageFacts, RuleTest, SaveRuleRequest, SiteMapRef } from "../../../data/types.js";
import { Button, SkeletonRows, StatusBadge, useDebounced } from "../../../ui/index.js";
import { PageSearch } from "../../findings/page-search.js";
import { conditionLabel, postTypeOption, schemaOption, verdictLabel, verdictTone } from "./labels.js";
import { asksAnything } from "./model.js";

const said = copy.classification.tester;
const facts = copy.classification.facts;

const draftDelayMs = 300;

export interface RuleTesterProps {
    siteId: string;
    rule: SaveRuleRequest;
    typeName: (typeId: string) => string;
    ruleName: (ruleId: string) => string;
}

function yesNo(value: boolean): string {
    return value ? facts.yes : facts.no;
}

function FactList({ held, paths }: { held: PageFacts; paths: ReadonlyMap<string, string> }): ReactElement {
    const ancestors = (held.ancestors ?? []).map((id) => paths.get(id) ?? id);
    const rows: readonly (readonly [string, string])[] = [
        [facts.path, held.path],
        [facts.slug, held.slug === "" ? facts.none : held.slug],
        [facts.postType, postTypeOption(held.postType)],
        [facts.level, String(held.level)],
        [facts.ancestors, ancestors.length === 0 ? facts.top : ancestors.join(" › ")],
        [facts.frontPage, yesNo(held.frontPage)],
        [facts.technical, held.technical === "" ? facts.notTechnical : facts.yes],
        [facts.rendered, yesNo(held.rendered)],
        [facts.listing, held.listing ? facts.listingOf(held.products) : facts.no],
        [facts.schema, (held.schema ?? []).length === 0 ? facts.none : (held.schema ?? []).map(schemaOption).join(", ")],
        [facts.words, held.words.toLocaleString("en-US")],
        [facts.h2, String(held.h2)],
    ];
    return (
        <dl className="grid grid-cols-[8.5rem_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs">
            {rows.map(([label, value]) => (
                <div key={label} className="contents">
                    <dt className="text-ink-dim">{label}</dt>
                    <dd className="min-w-0 truncate text-ink-soft" title={value}>
                        {value}
                    </dd>
                </div>
            ))}
        </dl>
    );
}

function outcomeOf(answered: RuleTest, typeName: (typeId: string) => string, ruleName: (ruleId: string) => string): string {
    const { outcome } = answered;
    if (outcome.waiting) {
        return said.outcomeWaiting;
    }
    const type = typeName(outcome.typeId);
    if (outcome.kept) {
        return said.outcomeKept(type);
    }
    if (outcome.reason === "unmatched") {
        return said.outcomeFallback(type);
    }
    if (outcome.ruleId !== "") {
        return said.outcomeRule(type, ruleName(outcome.ruleId));
    }
    return said.outcomeOther(type);
}

function headline(answered: RuleTest, path: string, type: string): string {
    switch (answered.verdict) {
        case "met":
            return said.verdict.met(path);
        case "waiting":
            return said.verdict.waiting(path);
        case "blocked":
            return said.verdict.blocked(path, type);
        default:
            return said.verdict.unmet(path);
    }
}

export function RuleTester({ siteId, rule, typeName, ruleName }: RuleTesterProps): ReactElement {
    const [page, setPage] = useState<SiteMapRef | null>(null);
    const settled = useDebounced(JSON.stringify(rule), draftDelayMs);
    const asking = page !== null && asksAnything(rule.conditions);
    const tested = useRuleTest(asking ? { siteId, pageId: page.pageId, rule: JSON.parse(settled) as SaveRuleRequest } : null);
    const answered = tested.data;
    const paths = usePagePaths(answered?.facts.ancestors ?? [], answered !== undefined);
    const problem = errorMessageOf(tested.error);

    return (
        <section aria-label={said.title} data-rule-tester={true} className="flex flex-col gap-2 rounded-md border border-hairline p-3">
            <div className="flex flex-col gap-0.5">
                <span className="text-sm font-semibold text-ink">{said.title}</span>
                <p className="text-xs text-ink-dim">{said.body}</p>
            </div>
            {page === null ? (
                <PageSearch siteId={siteId} picked={null} onPick={setPage} />
            ) : (
                <div className="flex flex-col gap-2.5">
                    <div className="flex items-center gap-2">
                        <span className="min-w-0 truncate font-mono text-xs text-ink-soft">{page.path}</span>
                        <Button
                            size="sm"
                            variant="ghost"
                            className="ml-auto"
                            onClick={() => {
                                setPage(null);
                            }}
                        >
                            {said.change}
                        </Button>
                    </div>
                    {!asking ? (
                        <p className="text-xs text-ink-dim">{said.noConditions}</p>
                    ) : problem !== null ? (
                        <p className="text-xs text-danger">{problem}</p>
                    ) : answered === undefined ? (
                        <SkeletonRows rows={3} label={said.checking} />
                    ) : (
                        <>
                            <p data-test-verdict={answered.verdict} className="text-sm text-ink">
                                {headline(answered, page.path, typeName(rule.typeId))}
                            </p>
                            <ul className="flex flex-col gap-1">
                                {(answered.conditions ?? []).map((held) => (
                                    <li key={held.condition} data-condition-verdict={held.condition} className="flex items-center justify-between gap-2 text-xs">
                                        <span className="text-ink-soft">{conditionLabel(held.condition)}</span>
                                        <StatusBadge tone={verdictTone(held.verdict)} dot={true}>
                                            {verdictLabel(held.verdict)}
                                        </StatusBadge>
                                    </li>
                                ))}
                            </ul>
                            <div className="flex flex-col gap-1.5">
                                <span className="text-xs font-medium text-ink-soft">{said.facts}</span>
                                <FactList held={answered.facts} paths={paths} />
                            </div>
                            <div className="flex flex-col gap-0.5">
                                <span className="text-xs font-medium text-ink-soft">{said.outcome}</span>
                                <p className="text-xs text-ink">{outcomeOf(answered, typeName, ruleName)}</p>
                            </div>
                        </>
                    )}
                </div>
            )}
        </section>
    );
}
