import type { ReactElement } from "react";
import { useEffect, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { failure, needsPlugin, pluginCodeOf, react } from "../../../data/errors.js";
import { usePreviewLink } from "../../../data/hooks/pages.js";
import { openExternal } from "../../../data/host.js";
import type { Page } from "../../../data/types.js";
import {
    Banner,
    Button,
    cx,
    DesktopWindowsIcon,
    Drawer,
    ExtensionOffIcon,
    MobileIcon,
    OpenInBrowserIcon,
    RefreshIcon,
    Segmented,
    SkeletonRows,
    StatusBadge,
    TabletIcon,
} from "../../../ui/index.js";
import type { SegmentedOption } from "../../../ui/index.js";
import type { Viewport } from "./model.js";
import { expiresInMinutes, frameScale, viewports } from "./model.js";

const drawerWidth = 1120;
const tickMs = 30_000;

const viewportOptions: readonly SegmentedOption<Viewport>[] = [
    { value: "desktop", label: copy.pages.preview.viewports.desktop, icon: DesktopWindowsIcon },
    { value: "tablet", label: copy.pages.preview.viewports.tablet, icon: TabletIcon },
    { value: "phone", label: copy.pages.preview.viewports.phone, icon: MobileIcon },
];

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

function useWidthOf(): [(node: HTMLDivElement | null) => void, number] {
    const [width, setWidth] = useState(0);
    const observer = useRef<ResizeObserver | null>(null);
    const attach = (node: HTMLDivElement | null): void => {
        observer.current?.disconnect();
        if (node === null) {
            return;
        }
        observer.current = new ResizeObserver((entries) => {
            const box = entries[0]?.contentRect;
            if (box !== undefined) {
                setWidth(box.width);
            }
        });
        observer.current.observe(node);
    };
    return [attach, width];
}

interface FrameProps {
    url: string;
    viewport: Viewport;
    title: string;
}

function Frame({ url, viewport, title }: FrameProps): ReactElement {
    const [attach, available] = useWidthOf();
    const [loaded, setLoaded] = useState(false);
    const width = viewports[viewport];
    const scale = frameScale(available - 32, width);

    useEffect(() => {
        setLoaded(false);
    }, [url]);

    return (
        <div ref={attach} className="relative flex min-h-0 flex-1 justify-center overflow-hidden bg-canvas p-4">
            <div
                className="relative origin-top overflow-hidden rounded-md border border-edge bg-white"
                style={{ width: `${width}px`, height: `calc(100% / ${scale})`, transform: `scale(${scale})` }}
            >
                {loaded ? null : (
                    <div className="absolute inset-0 bg-panel p-4">
                        <SkeletonRows rows={12} label={copy.pages.preview.loading} />
                    </div>
                )}
                <iframe
                    key={url}
                    src={url}
                    title={title}
                    sandbox="allow-scripts allow-same-origin allow-forms"
                    referrerPolicy="no-referrer"
                    onLoad={() => {
                        setLoaded(true);
                    }}
                    className={cx("h-full w-full border-0", loaded ? "opacity-100" : "opacity-0")}
                />
            </div>
        </div>
    );
}

export interface SitePreviewDrawerProps {
    page: Page;
    editorUrl: string | null;
    onClose: () => void;
}

export function SitePreviewDrawer({ page, editorUrl, onClose }: SitePreviewDrawerProps): ReactElement {
    const said = copy.pages.preview;
    const link = usePreviewLink(page);
    const now = useNow();
    const [viewport, setViewport] = useState<Viewport>("desktop");

    const answer = link.data;
    const minutes = answer === undefined ? null : expiresInMinutes(answer.expiresAt, now);
    const expired = minutes === 0;
    const pluginCode = link.error === null ? null : pluginCodeOf(failure(link.error));
    const reaction = link.error === null ? null : react(link.error);

    const header = (
        <div className="flex items-center gap-2">
            {answer === undefined ? null : answer.kind === "public" ? (
                <StatusBadge tone="ok">{said.public}</StatusBadge>
            ) : (
                <StatusBadge tone={expired ? "warn" : "info"}>
                    {said.draft}
                    {minutes === null ? null : <span className="font-mono font-normal"> · {said.expiresIn(minutes)}</span>}
                </StatusBadge>
            )}
            <Segmented
                label={said.viewport}
                value={viewport}
                options={viewportOptions}
                onValueChange={setViewport}
                iconOnly={true}
            />
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
                icon={OpenInBrowserIcon}
                disabled={answer === undefined || expired}
                onClick={() => {
                    if (answer !== undefined) {
                        void openExternal(answer.url);
                    }
                }}
            >
                {said.openInBrowser}
            </Button>
        </div>
    );

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={said.title(page.path)}
            closeLabel={said.close}
            width={drawerWidth}
            header={header}
        >
            <div className="flex h-full min-h-0 flex-col">
                {link.isPending ? (
                    <div className="p-4">
                        <SkeletonRows rows={10} label={said.loading} />
                    </div>
                ) : answer !== undefined ? (
                    <>
                        <p className="shrink-0 border-b border-hairline px-3 py-1.5 text-2xs text-ink-faint">
                            {answer.kind === "public" ? said.publicBody : said.draftBody} {said.embedHint}
                        </p>
                        {expired ? (
                            <div className="p-4">
                                <Banner tone="warn" title={said.expired} />
                            </div>
                        ) : (
                            <Frame url={answer.url} viewport={viewport} title={said.title(page.path)} />
                        )}
                    </>
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
                                        icon={OpenInBrowserIcon}
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
                            title={reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? copy.errors.INTERNAL : reaction.message}
                        />
                    </div>
                )}
            </div>
        </Drawer>
    );
}
