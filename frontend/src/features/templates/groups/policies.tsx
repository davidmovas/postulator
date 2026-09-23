import type { ReactElement } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { useSite } from "../../../data/hooks/sites.js";
import { useEffectivePolicy } from "../../../data/hooks/templates.js";
import { Banner, Button, SkeletonRows, StatusBadge } from "../../../ui/index.js";
import { anchorLabel, flagLabel, scopeLabel, scopeTone } from "../labels.js";

export interface PoliciesGroupProps {
    siteId: string;
}

export function PoliciesGroup({ siteId }: PoliciesGroupProps): ReactElement {
    const navigate = useNavigate();
    const site = useSite(siteId === "" ? null : siteId);
    const effective = useEffectivePolicy(siteId === "" ? null : siteId);
    const policy = effective.data?.policy;
    const chosen = site.data?.site.defaults.linkPolicyId ?? null;

    if (effective.isPending) {
        return <SkeletonRows rows={4} label={copy.policies.loading} />;
    }

    if (policy === undefined) {
        return (
            <Banner
                tone="warn"
                title={
                    failure(effective.error).code === "NOT_FOUND"
                        ? copy.policies.effectiveMissing
                        : failure(effective.error).message
                }
                actions={
                    <Button
                        onClick={() => {
                            void navigate(`/s/${siteId}/templates/policies`);
                        }}
                    >
                        {copy.policies.newPolicy}
                    </Button>
                }
            />
        );
    }

    return (
        <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
            <header className="flex h-10 items-center justify-between gap-3 border-b border-hairline px-3">
                <div className="flex min-w-0 items-center gap-2">
                    <span className="truncate text-sm font-semibold text-ink">{policy.name}</span>
                    <StatusBadge tone={scopeTone(policy.scope)} dot={false}>
                        {scopeLabel(policy.scope)}
                    </StatusBadge>
                </div>
                <Button
                    size="sm"
                    onClick={() => {
                        void navigate(`/s/${siteId}/templates/policies`);
                    }}
                >
                    {copy.policies.title}
                </Button>
            </header>
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">
                    {chosen === policy.id ? copy.policies.effectiveFromSite : copy.policies.effectiveFallback}
                </p>
                <dl className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 text-xs">
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.forbidExternal}</dt>
                    <dd className="text-ink">{flagLabel(policy.forbidExternal)}</dd>
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.forbidSelf}</dt>
                    <dd className="text-ink">{flagLabel(policy.forbidSelf)}</dd>
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.anchorStrategy}</dt>
                    <dd className="text-ink">{anchorLabel(policy.anchorStrategy)}</dd>
                </dl>
            </div>
        </section>
    );
}
