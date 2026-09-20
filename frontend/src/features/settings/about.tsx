import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { useBuildInfo } from "../../data/hooks/health.js";
import { useModelCatalog } from "../../data/hooks/models.js";
import { useProviderKeys, useSettingsSchema } from "../../data/hooks/settings.js";
import { useLockGate } from "../../data/lock.js";
import { useTools } from "../../data/hooks/tools.js";
import { absoluteTime } from "../../domain/format.js";
import { Button, Panel, PanelHeader, RefreshIcon, Skeleton } from "../../ui/index.js";

const unknownBuild = "unknown";

function Row({ label, value }: { label: string; value: string }): ReactElement {
    return (
        <div className="flex items-baseline justify-between gap-4 px-3 py-1.5">
            <span className="text-xs text-ink-dim">{label}</span>
            <span className="truncate font-mono text-xs text-ink">{value}</span>
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
    const schema = useSettingsSchema();
    const catalog = useModelCatalog();
    const providers = useProviderKeys();
    const tools = useTools();
    const gate = useLockGate();

    const configured = (providers.data?.providers ?? []).filter((provider) => provider.configured).length;

    return (
        <div className="flex max-w-2xl flex-col gap-3">
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
                <div className="flex flex-col gap-1 p-3">
                    <p className="text-sm text-ink-soft">{copy.app.tagline}</p>
                    {build.data === undefined ? (
                        <Skeleton height={72} />
                    ) : (
                        <div className="mt-1 divide-y divide-hairline rounded-md border border-hairline">
                            <Row label={copy.settings.about.version} value={build.data.version} />
                            <Row label={copy.settings.about.commit} value={build.data.commit} />
                            <Row label={copy.settings.about.built} value={builtAt(build.data.buildDate)} />
                        </div>
                    )}
                </div>
            </Panel>

            <Panel>
                <PanelHeader title={copy.settings.about.carries.title} />
                <div className="divide-y divide-hairline">
                    <Row label={copy.settings.about.carries.settings} value={String(schema.data?.settings?.length ?? 0)} />
                    <Row label={copy.settings.about.carries.models} value={String(catalog.data?.models?.length ?? 0)} />
                    <Row label={copy.settings.about.carries.providers} value={String(configured)} />
                    <Row label={copy.settings.about.carries.tools} value={String(tools.data?.tools?.length ?? 0)} />
                </div>
                <div className="border-t border-hairline px-3 py-2">
                    <Link to="/settings/security" className="text-xs text-accent hover:underline">
                        {gate.protectedByPassword ? copy.shell.unlocked : copy.shell.unprotected}
                    </Link>
                </div>
            </Panel>
        </div>
    );
}
