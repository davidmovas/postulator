import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { failure, needsPlugin, pluginCodeOf, react } from "../../../data/errors.js";
import { usePreviewLink } from "../../../data/hooks/pages.js";
import { useSite } from "../../../data/hooks/sites.js";
import { openExternal } from "../../../data/host.js";
import type { Page } from "../../../data/types.js";
import {
    Banner,
    Button,
    DesktopWindowsIcon,
    EmptyState,
    ExtensionOffIcon,
    MobileIcon,
    OpenInNewIcon,
    PreviewIcon,
    PublicIcon,
    RefreshIcon,
    Segmented,
    SkeletonRows,
    StatusBadge,
    TabletIcon,
} from "../../../ui/index.js";
import type { SegmentedOption } from "../../../ui/index.js";
import type { PreviewState, Viewport } from "./model.js";
import { editorPreviewUrl, expiresInMinutes, previewState } from "./model.js";
import { PreviewFrame } from "./frame.js";

const tickMs = 30_000;

const viewportOptions: readonly SegmentedOption<Viewport>[] = [
    { value: "desktop", label: copy.pages.preview.viewports.desktop, icon: DesktopWindowsIcon },
    { value: "tablet", label: copy.pages.preview.viewports.tablet, icon: TabletIcon },
    { value: "phone", label: copy.pages.preview.viewports.phone, icon: MobileIcon },
];

const blocked: Readonly<Record<string, { title: string; body: string }>> = {
    "draft-needs-plugin": { title: copy.pages.preview.needsPlugin, body: copy.pages.preview.needsPluginBody },
    "draft-plugin-outdated": {
        title: copy.pages.preview.outdatedPlugin,
        body: copy.pages.preview.outdatedPluginBody,
    },
    "not-on-site": { title: copy.pages.preview.notOnSite, body: copy.pages.preview.notOnSiteBody },
    archived: { title: copy.pages.preview.archived, body: copy.pages.preview.archivedBody },
};

function useNow(): number {
    const [now, setNow] = useState(() => Date.now());
    useEffect(() => {
        const timer = window.setInterval(() => {
            setNow(Date.now());
        }, tickMs);
        return () => {
            window.clearInterval(timer);
        };
    }, []);
    return now;
}

function iconOf(state: PreviewState): typeof PreviewIcon {
    if (state === "draft-needs-plugin" || state === "draft-plugin-outdated") {
        return ExtensionOffIcon;
    }
    return state === "public" ? PublicIcon : PreviewIcon;
}

export interface PreviewTabProps {
    page: Page;
    siteId: string;
}

export function PreviewTab({ page, siteId }: PreviewTabProps): ReactElement {
    const said = copy.pages.preview;
    const site = useSite(siteId);
    const link = usePreviewLink(page);
    const now = useNow();
    const [viewport, setViewport] = useState<Viewport>("desktop");

    const state = previewState({ status: page.status, wpId: page.wpId }, site.data?.site.plugin ?? null);
    const editorUrl = editorPreviewUrl(site.data?.site.baseUrl ?? "", page.wpType, page.wpId);
    const answer = link.data;
    const minutes = answer === undefined ? null : expiresInMinutes(answer.expiresAt, now);
    const expired = minutes === 0;
    const pluginCode = link.error === null ? null : pluginCodeOf(failure(link.error));
    const reaction = link.error === null ? null : react(link.error);
    const stop = blocked[state];

    if (stop !== undefined) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center p-6">
                <EmptyState
                    icon={iconOf(state)}
                    title={stop.title}
                    body={stop.body}
                    className="w-96"
                    actions={
                        editorUrl === null ? undefined : (
                            <Button
                                icon={OpenInNewIcon}
                                onClick={() => {
                                    void openExternal(editorUrl);
                                }}
                            >
                                {copy.pages.openOnSite}
                            </Button>
                        )
                    }
                />
            </div>
        );
    }

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="flex h-9 shrink-0 items-center gap-2 border-b border-hairline px-3">
                <Segmented
                    label={said.viewport}
                    value={viewport}
                    options={viewportOptions}
                    onValueChange={setViewport}
                    iconOnly={true}
                />
                {answer === undefined ? null : answer.kind === "public" ? (
                    <StatusBadge tone="ok">{said.public}</StatusBadge>
                ) : (
                    <StatusBadge tone={expired ? "warn" : "info"}>
                        {minutes === null ? said.draft : said.expiresIn(minutes)}
                    </StatusBadge>
                )}
                <div className="ml-auto flex shrink-0 items-center gap-2">
                    {answer?.kind === "preview" ? (
                        <Button
                            size="sm"
                            variant="ghost"
                            icon={RefreshIcon}
                            busy={link.isFetching}
                            onClick={() => {
                                void link.refetch();
                            }}
                        >
                            {said.newLink}
                        </Button>
                    ) : null}
                    <Button
                        size="sm"
                        icon={OpenInNewIcon}
                        disabled={answer === undefined || expired}
                        title={expired ? said.expired : copy.pages.openOnSite}
                        onClick={() => {
                            if (answer !== undefined) {
                                void openExternal(answer.url);
                            }
                        }}
                    >
                        {copy.pages.openOnSite}
                    </Button>
                </div>
            </div>
            {link.isPending ? (
                <div className="p-4">
                    <SkeletonRows rows={10} label={said.loading} />
                </div>
            ) : answer !== undefined ? (
                expired ? (
                    <div className="p-4">
                        <Banner tone="warn" title={said.expired} />
                    </div>
                ) : (
                    <PreviewFrame url={answer.url} viewport={viewport} title={said.title(page.path)} />
                )
            ) : needsPlugin(link.error) ? (
                <div className="p-4">
                    <Banner
                        tone="info"
                        icon={ExtensionOffIcon}
                        title={pluginCode === "plugin_outdated" ? said.outdatedPlugin : said.needsPlugin}
                        body={pluginCode === "plugin_outdated" ? said.outdatedPluginBody : said.needsPluginBody}
                        actions={
                            editorUrl === null ? undefined : (
                                <Button
                                    size="sm"
                                    icon={OpenInNewIcon}
                                    onClick={() => {
                                        void openExternal(editorUrl);
                                    }}
                                >
                                    {said.openAnyway}
                                </Button>
                            )
                        }
                    />
                </div>
            ) : (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={
                            reaction === null || reaction.kind === "silent" || reaction.kind === "unlock"
                                ? copy.errors.INTERNAL
                                : reaction.message
                        }
                    />
                </div>
            )}
        </div>
    );
}
