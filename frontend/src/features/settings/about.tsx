import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useBuildInfo } from "../../data/hooks/health.js";
import { absoluteTime } from "../../domain/format.js";
import { Button, Panel, PanelHeader, RefreshIcon, Skeleton } from "../../ui/index.js";

const unknownBuild = "unknown";

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
        </div>
    );
}
