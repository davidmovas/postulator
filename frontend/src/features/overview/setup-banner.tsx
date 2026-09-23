import type { ReactElement } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { useSavePluginPackage } from "../../data/hooks/sync.js";
import { pickSaveFile } from "../../data/host.js";
import { pushToast } from "../../data/toasts.js";
import { Banner, Button, ExtensionOffIcon } from "../../ui/index.js";

export interface SetupBannerProps {
    onDismiss: () => void;
}

export function SetupBanner({ onDismiss }: SetupBannerProps): ReactElement {
    const navigate = useNavigate();
    const save = useSavePluginPackage();

    const download = async (): Promise<void> => {
        const path = await pickSaveFile({
            title: copy.overview.plugin.save,
            filename: copy.overview.plugin.fileName,
            filters: [{ displayName: copy.overview.plugin.filter, pattern: "*.zip" }],
        });
        if (path === null) {
            return;
        }
        await save.mutateAsync({ path });
        pushToast("info", copy.overview.plugin.saved);
    };

    return (
        <Banner
            tone="warn"
            icon={ExtensionOffIcon}
            title={copy.overview.plugin.title}
            body={copy.overview.plugin.body}
            actions={
                <>
                    <Button
                        variant="primary"
                        busy={save.isPending}
                        onClick={() => {
                            void download();
                        }}
                    >
                        {copy.overview.plugin.save}
                    </Button>
                    <Button
                        onClick={() => {
                            void navigate("/sites");
                        }}
                    >
                        {copy.overview.plugin.open}
                    </Button>
                    <Button variant="ghost" onClick={onDismiss}>
                        {copy.overview.plugin.dismiss}
                    </Button>
                </>
            }
        />
    );
}
