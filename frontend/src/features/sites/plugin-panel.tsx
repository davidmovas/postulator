import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { usePluginState, useSavePluginPackage } from "../../data/hooks/sync.js";
import { pickSaveFile } from "../../data/host.js";
import { pushToast } from "../../data/toasts.js";
import type { Site } from "../../data/types.js";
import {
    Banner,
    Button,
    CloudUploadIcon,
    ExtensionIcon,
    ExtensionOffIcon,
    Skeleton,
    SyncIcon,
} from "../../ui/index.js";

const packageFilename = "postulator-companion.zip";

export interface PluginPanelProps {
    site: Site;
}

export function PluginPanel({ site }: PluginPanelProps): ReactElement {
    const state = usePluginState(site.id);
    const save = useSavePluginPackage();

    const plugin = state.data?.plugin ?? null;

    const download = (): void => {
        void (async () => {
            const path = await pickSaveFile({
                title: copy.sites.plugin.download,
                filename: packageFilename,
                filters: [{ displayName: "Zip archive", pattern: "*.zip" }],
            });
            if (path === null) {
                return;
            }
            save.mutate(
                { path },
                {
                    onSuccess: (answered) => {
                        pushToast("info", copy.sites.plugin.saved(answered.path));
                    },
                },
            );
        })();
    };

    const recheck = (
        <Button
            size="sm"
            variant="secondary"
            icon={SyncIcon}
            busy={state.isFetching}
            onClick={() => {
                void state.refetch();
            }}
        >
            {copy.sites.plugin.recheck}
        </Button>
    );

    const downloadButton = (
        <Button
            size="sm"
            variant="secondary"
            icon={CloudUploadIcon}
            busy={save.isPending}
            onClick={download}
        >
            {copy.sites.plugin.download}
        </Button>
    );

    if (plugin === null) {
        return <Skeleton height={64} />;
    }

    if (plugin.installed) {
        const listed = plugin.capabilities ?? [];
        const capabilities = listed.length === 0 ? copy.sites.plugin.noCapabilities : listed.join(", ");
        const seo = plugin.seoPlugin === "" ? copy.sites.plugin.noSeo : plugin.seoPlugin;
        return (
            <Banner
                tone="ok"
                icon={ExtensionIcon}
                title={copy.sites.plugin.installed(plugin.version)}
                body={`${copy.sites.plugin.capabilities}: ${capabilities} · ${copy.sites.plugin.seo}: ${seo}`}
                actions={recheck}
            />
        );
    }

    return (
        <Banner
            tone="warn"
            icon={ExtensionOffIcon}
            title={copy.sites.plugin.missingTitle(site.name)}
            body={copy.sites.plugin.missingBody}
            actions={
                <>
                    <ul className="w-full list-disc pl-4 text-xs text-ink-soft">
                        {copy.sites.plugin.losses.map((loss) => (
                            <li key={loss}>{loss}</li>
                        ))}
                    </ul>
                    {recheck}
                    {downloadButton}
                </>
            }
        />
    );
}
