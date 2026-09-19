import type { KeyboardEvent } from "react";
import { Outlet, useLocation } from "react-router";

import { copy } from "../copy/index.js";
import { dismissToast, useToasts } from "../data/toasts.js";
import type { ToastTone } from "../data/toasts.js";
import { AgentDock, toggleDock, useDock } from "../features/agent/index.js";
import { IconButton, RightPanelOpenIcon, Toast, ToastRegion } from "../ui/index.js";
import type { Tone } from "../ui/index.js";
import { Rail } from "./rail.js";
import { StatusBar } from "./statusbar.js";
import { TitleBar } from "./titlebar.js";

const toastTone: Readonly<Record<ToastTone, Tone>> = {
    danger: "danger",
    warning: "warn",
    info: "info",
};

export function siteIdOf(pathname: string): string | null {
    const matched = /^\/s\/([^/]+)/.exec(pathname);
    return matched === null ? null : matched[1];
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

function shortcut(event: KeyboardEvent<HTMLDivElement>): void {
    if ((event.ctrlKey || event.metaKey) && !event.altKey && event.key.toLowerCase() === "j") {
        event.preventDefault();
        toggleDock();
    }
}

export function Shell() {
    const location = useLocation();
    const siteId = siteIdOf(location.pathname);
    const dock = useDock();

    return (
        <div className="flex h-full flex-col bg-canvas" onKeyDown={shortcut}>
            <TitleBar siteId={siteId} />
            <div className="flex min-h-0 flex-1">
                <Rail siteId={siteId} />
                <main className="min-w-0 flex-1 overflow-auto">
                    <Outlet />
                </main>
                {dock.open ? (
                    <AgentDock siteId={siteId} />
                ) : (
                    <div className="flex w-8 shrink-0 justify-center border-l border-hairline bg-panel pt-2">
                        <IconButton
                            icon={RightPanelOpenIcon}
                            label={copy.shell.expandDock}
                            variant="ghost"
                            size="sm"
                            onClick={toggleDock}
                        />
                    </div>
                )}
            </div>
            <StatusBar siteId={siteId} />
            <Toasts />
        </div>
    );
}
