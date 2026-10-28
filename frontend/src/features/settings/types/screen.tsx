import type { ReactElement } from "react";
import { useState } from "react";
import { Link, Navigate, useSearchParams } from "react-router";

import { readLastSite } from "../../../app/site-memory.js";
import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { useSites } from "../../../data/hooks/sites.js";
import type { SegmentedOption } from "../../../ui/index.js";
import { Banner, Button, EmptyState, HistoryIcon, Panel, PreviewIcon, PublicIcon, Segmented, Select, SkeletonRows } from "../../../ui/index.js";
import { useRuleNaming } from "../classification/naming.js";
import { ClassificationPreview } from "../classification/preview.js";
import { RulesView } from "../classification/rules.js";
import { pageTypeRulesOf, rulesView } from "../tabs.js";
import { TypesView } from "./cards.js";
import { pickSite } from "./model.js";
import { TypeSteps } from "./steps.js";

const said = copy.pageTypes.screen;

type View = "types" | "rules";

const viewOptions: readonly SegmentedOption<View>[] = [
    { value: "types", label: said.views.types },
    { value: "rules", label: said.views.rules },
];

function viewOf(asked: string | null): View {
    return asked === rulesView ? "rules" : "types";
}

interface SitePageTypesProps {
    siteId: string;
    view: View;
    typeId: string | null;
    onOpen: (typeId: string | null) => void;
}

function SitePageTypes({ siteId, view, typeId, onOpen }: SitePageTypesProps): ReactElement {
    const naming = useRuleNaming(siteId);
    const [previewing, setPreviewing] = useState(false);
    const preview = (): void => {
        setPreviewing(true);
    };
    const before = naming.beforeRules;

    return (
        <>
            <TypeSteps siteId={siteId} />
            {before === 0 ? null : (
                <div data-before-rules={before}>
                    <Banner
                        tone="info"
                        icon={HistoryIcon}
                        title={said.before(before)}
                        body={said.beforeBody(before)}
                        actions={
                            <Button size="sm" icon={PreviewIcon} onClick={preview}>
                                {said.beforeAction}
                            </Button>
                        }
                    />
                </div>
            )}
            {view === "rules" ? (
                <RulesView siteId={siteId} naming={naming} onPreview={preview} />
            ) : (
                <TypesView siteId={siteId} naming={naming} typeId={typeId} onOpen={onOpen} />
            )}
            <ClassificationPreview
                open={previewing}
                onOpenChange={setPreviewing}
                siteId={siteId}
                fallbackType={naming.fallback === null ? copy.classification.lostType : naming.typeName(naming.fallback.typeId)}
                typeName={naming.typeName}
                ruleName={naming.ruleName}
            />
        </>
    );
}

export function PageTypesScreen(): ReactElement {
    const [searchParams, setSearchParams] = useSearchParams();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const siteId = pickSite(searchParams.get("site"), readLastSite(), rows.map((site) => site.id));
    const view = viewOf(searchParams.get("view"));
    const typeId = searchParams.get("type");

    const go = (site: string, nextView: View, type: string | null): void => {
        const next = new URLSearchParams({ site });
        if (nextView === "rules") {
            next.set("view", rulesView);
        }
        if (type !== null) {
            next.set("type", type);
        }
        setSearchParams(next, { replace: true });
    };

    return (
        <div className="mx-auto flex w-full max-w-5xl flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <Segmented
                    label={said.viewsLabel}
                    value={view}
                    options={viewOptions}
                    onValueChange={(next) => {
                        if (siteId !== null) {
                            go(siteId, next, null);
                        }
                    }}
                />
                {siteId === null ? null : (
                    <div className="w-64 max-w-full">
                        <Select
                            aria-label={said.site}
                            value={siteId}
                            options={rows.map((site) => ({ value: site.id, label: site.name }))}
                            onValueChange={(site) => {
                                go(site, view, null);
                            }}
                        />
                    </div>
                )}
            </div>
            {sites.isPending ? (
                <Panel>
                    <SkeletonRows rows={4} label={said.loading} className="p-3" />
                </Panel>
            ) : siteId === null ? (
                <Panel>
                    <div className="p-3">
                        <EmptyState
                            icon={PublicIcon}
                            title={said.noSites}
                            actions={
                                <Link to="/sites">
                                    <Button size="sm">{said.openSites}</Button>
                                </Link>
                            }
                        />
                    </div>
                </Panel>
            ) : (
                <SitePageTypes
                    key={siteId}
                    siteId={siteId}
                    view={view}
                    typeId={typeId}
                    onOpen={(type) => {
                        go(siteId, view, type);
                    }}
                />
            )}
        </div>
    );
}

export function RulesRedirect(): ReactElement {
    const [searchParams] = useSearchParams();
    return <Navigate to={pageTypeRulesOf(searchParams.get("site"))} replace={true} />;
}
