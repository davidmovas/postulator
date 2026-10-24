import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { usePluginState, useSavePluginPackage } from "../../data/hooks/sync.js";
import { pickSaveFile } from "../../data/host.js";
import { pushToast } from "../../data/toasts.js";
import type { Site } from "../../data/types.js";
import {
    Banner,
    Button,
    CategoryIcon,
    CloudUploadIcon,
    ExtensionIcon,
    ExtensionOffIcon,
    LockIcon,
    Skeleton,
    SyncIcon,
} from "../../ui/index.js";
import { pluginBanner, storeBanner } from "./plugin.js";

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

    const store = storeBanner(state.data?.commerce ?? site.commerce, plugin.installed, site.username);
    const storePanel =
        store === null ? null : (
            <Banner
                tone={store.tone}
                icon={store.tone === "ok" ? CategoryIcon : LockIcon}
                title={store.title}
                body={store.body}
            />
        );

    const banner = pluginBanner(plugin, site.name);
    return (
        <>
            <Banner
                tone={banner.tone}
                icon={plugin.installed ? ExtensionIcon : ExtensionOffIcon}
                title={banner.title}
                body={banner.body}
                actions={
                    banner.offersZip ? (
                        <>
                            {recheck}
                            {downloadButton}
                        </>
                    ) : (
                        recheck
                    )
                }
            />
            {storePanel}
        </>
    );
}
