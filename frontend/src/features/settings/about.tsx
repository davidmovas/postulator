import type { ReactElement } from "react";
import { useState } from "react";

import { readLastSite } from "../../app/site-memory.js";
import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useBuildInfo } from "../../data/hooks/health.js";
import { useSites } from "../../data/hooks/sites.js";
import { bundleRefusal, useSaveBundle } from "../../data/hooks/support.js";
import { absoluteTime, bytes } from "../../domain/format.js";
import { Banner, Button, Inventory2Icon, Panel, PanelHeader, RefreshIcon, Select, Skeleton } from "../../ui/index.js";
import { pickSite } from "./types/model.js";

const unknownBuild = "unknown";

const said = copy.settings.about.support;

function SupportPanel(): ReactElement {
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const [chosen, setChosen] = useState<string | null>(null);
    const siteId = pickSite(chosen, readLastSite(), rows.map((site) => site.id));
    const site = rows.find((held) => held.id === siteId) ?? null;
    const save = useSaveBundle();
    const saved = save.data ?? null;
    const refusal = bundleRefusal(save.error);

    return (
        <Panel>
            <PanelHeader title={said.title} />
            <div className="flex flex-col gap-2 p-3" data-support-bundle={true}>
                {sites.isPending ? (
                    <Skeleton height={28} />
                ) : site === null ? (
                    <p className="text-xs text-ink-dim">{said.noSites}</p>
                ) : (
                    <div className="flex flex-wrap items-center gap-2">
                        {rows.length > 1 ? (
                            <div className="w-56">
                                <Select
                                    size="sm"
                                    aria-label={said.site}
                                    value={site.id}
                                    options={rows.map((held) => ({ value: held.id, label: held.name }))}
                                    onValueChange={(next) => {
                                        setChosen(next);
                                        save.reset();
                                    }}
                                />
                            </div>
                        ) : null}
                        <Button
                            variant="secondary"
                            icon={Inventory2Icon}
                            busy={save.isPending}
                            onClick={() => {
                                save.mutate({ siteId: site.id, siteName: site.name });
                            }}
                        >
                            {save.isPending ? said.saving : said.save}
                        </Button>
                    </div>
                )}
                <p className="text-xs text-ink-dim">{said.holds}</p>
                {saved === null ? null : <Banner tone="ok" title={said.saved(saved.path, bytes(saved.bytes))} />}
                {refusal === null ? null : <Banner tone="danger" title={refusal} />}
            </div>
        </Panel>
    );
}

function Row({ label, value }: { label: string; value: string }): ReactElement {
    return (
        <div className="flex items-baseline justify-between gap-4 border-b border-hairline px-3 py-2 last:border-b-0">
            <span className="text-sm text-ink-soft">{label}</span>
            <span className="min-w-0 truncate font-mono text-xs text-ink">{value}</span>
        </div>
    );
}

function builtAt(value: string): string {
    if (value === "" || value === unknownBuild) {
        return unknownBuild;
    }
    const rendered = absoluteTime(value);
    return rendered === "never" ? value : rendered;
}

export function AboutScreen(): ReactElement {
    const build = useBuildInfo();

    return (
        <div className="mx-auto flex w-full max-w-lg flex-col gap-4">
            <Panel>
                <PanelHeader title={copy.settings.about.title}>
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={RefreshIcon}
                        busy={build.isFetching}
                        onClick={() => {
                            void build.refetch();
                        }}
                    >
                        {copy.settings.about.recheck}
                    </Button>
                </PanelHeader>
                {build.data === undefined ? (
                    <div className="p-3">
                        <Skeleton height={72} />
                    </div>
                ) : (
                    <>
                        <Row label={copy.settings.about.version} value={build.data.version} />
                        <Row label={copy.settings.about.commit} value={build.data.commit} />
                        <Row label={copy.settings.about.built} value={builtAt(build.data.buildDate)} />
                    </>
                )}
            </Panel>
            <SupportPanel />
        </div>
    );
}
