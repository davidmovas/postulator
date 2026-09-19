import { useState } from "react";
import { Outlet, useLocation } from "react-router";

import { copy } from "../copy/index.js";
import { dismissToast, useToasts } from "../data/toasts.js";
import type { ToastTone } from "../data/toasts.js";
import { IconButton, RightPanelCloseIcon, RightPanelOpenIcon, Toast, ToastRegion } from "../ui/index.js";
import type { Tone } from "../ui/index.js";
import { NotBuilt } from "./not-built.js";
import { Rail } from "./rail.js";
import { StatusBar } from "./statusbar.js";
import { TitleBar } from "./titlebar.js";

const dockWidthKey = "postulator.dock.width";
const minimumDockWidth = 280;
const maximumDockWidth = 640;

const toastTone: Readonly<Record<ToastTone, Tone>> = {
    danger: "danger",
    warning: "warn",
    info: "info",
};

export function siteIdOf(pathname: string): string | null {
    const matched = /^\/s\/([^/]+)/.exec(pathname);
    return matched === null ? null : matched[1];
}

function storedDockWidth(): number {
    const held = window.localStorage.getItem(dockWidthKey);
    const parsed = held === null ? Number.NaN : Number.parseInt(held, 10);
    if (Number.isNaN(parsed)) {
        return 340;
    }
    return Math.min(Math.max(parsed, minimumDockWidth), maximumDockWidth);
}

function Toasts() {
    const toasts = useToasts();
    if (toasts.length === 0) {
        return null;
    }
    return (
        <ToastRegion>
            {toasts.map((toast) => (
                <Toast
                    key={toast.id}
                    tone={toastTone[toast.tone]}
                    message={toast.message}
                    dismissLabel={copy.app.dismiss}
                    onDismiss={() => {
                        dismissToast(toast.id);
                    }}
                />
            ))}
        </ToastRegion>
    );
}

export function Shell() {
    const location = useLocation();
    const siteId = siteIdOf(location.pathname);
    const [dockOpen, setDockOpen] = useState(false);
    const [dockWidth, setDockWidth] = useState(storedDockWidth);

    const resize = (next: number): void => {
        const clamped = Math.min(Math.max(next, minimumDockWidth), maximumDockWidth);
        setDockWidth(clamped);
        window.localStorage.setItem(dockWidthKey, String(clamped));
    };

    return (
        <div className="flex h-full flex-col bg-canvas">
            <TitleBar siteId={siteId} />
            <div className="flex min-h-0 flex-1">
                <Rail siteId={siteId} />
                <main className="min-w-0 flex-1 overflow-auto">
                    <Outlet />
                </main>
                {dockOpen ? (
                    <aside
                        aria-label={copy.shell.agentDock}
                        className="flex shrink-0 flex-col border-l border-hairline bg-panel"
                        style={{ width: `${dockWidth}px` }}
                    >
                        <header className="flex h-8 shrink-0 items-center justify-between gap-2 border-b border-hairline px-2">
                            <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">
                                {copy.shell.agentDock}
                            </span>
                            <IconButton
                                icon={RightPanelCloseIcon}
                                label={copy.shell.collapseDock}
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    setDockOpen(false);
                                }}
                            />
                        </header>
                        <input
                            type="range"
                            aria-label={copy.shell.agentDock}
                            className="mx-2 mt-2 accent-accent"
                            min={minimumDockWidth}
                            max={maximumDockWidth}
                            value={dockWidth}
                            onChange={(event) => {
                                resize(Number.parseInt(event.target.value, 10));
                            }}
                        />
                        <div className="min-h-0 flex-1 overflow-auto">
                            <NotBuilt screen={copy.shell.agentDock} wave="wave 3, agent 6" />
                        </div>
                    </aside>
                ) : (
                    <div className="flex w-8 shrink-0 justify-center border-l border-hairline bg-panel pt-2">
                        <IconButton
                            icon={RightPanelOpenIcon}
                            label={copy.shell.expandDock}
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                                setDockOpen(true);
                            }}
                        />
                    </div>
                )}
            </div>
            <StatusBar siteId={siteId} />
            <Toasts />
        </div>
    );
}
