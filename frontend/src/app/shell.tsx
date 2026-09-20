import { useEffect, useMemo } from "react";
import { Link, Outlet, useLocation } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { usePendingActions } from "../data/hooks/agent.js";
import { dismissToast, useToasts } from "../data/toasts.js";
import type { ToastTone } from "../data/toasts.js";
import { AgentDock, toggleDock, useDock } from "../features/agent/index.js";
import { CommandPalette, openPalette } from "../features/palette/index.js";
import { Button, Toast, ToastRegion } from "../ui/index.js";
import type { Tone } from "../ui/index.js";
import { goToEntries } from "./nav.js";
import { Rail } from "./rail.js";
import { rememberSite } from "./site-memory.js";
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
                    action={
                        toast.action === null ? undefined : (
                            <Link to={toast.action.to}>
                                <Button
                                    size="sm"
                                    onClick={() => {
                                        dismissToast(toast.id);
                                    }}
                                >
                                    {toast.action.label}
                                </Button>
                            </Link>
                        )
                    }
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
    const dock = useDock();
    const pending = usePendingActions({ status: "pending" }, 100);
    const awaiting = flatten(pending.data?.pages).length;
    const destinations = useMemo(() => goToEntries(siteId), [siteId]);

    useEffect(() => {
        if (siteId !== null) {
            rememberSite(siteId);
        }
    }, [siteId]);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent): void => {
            if (!event.ctrlKey && !event.metaKey) {
                return;
            }
            if (event.altKey) {
                return;
            }
            const pressed = event.key.toLowerCase();
            if (pressed === "k") {
                event.preventDefault();
                openPalette();
                return;
            }
            if (pressed === "j") {
                event.preventDefault();
                toggleDock();
            }
        };
        window.addEventListener("keydown", onKeyDown);
        return () => {
            window.removeEventListener("keydown", onKeyDown);
        };
    }, []);

    return (
        <div className="flex h-full flex-col bg-canvas">
            <TitleBar siteId={siteId} dockOpen={dock.open} onToggleDock={toggleDock} />
            <div className="flex min-h-0 flex-1">
                <Rail siteId={siteId} pending={awaiting} />
                <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
                    <Outlet />
                </main>
                {dock.open ? <AgentDock siteId={siteId} /> : null}
            </div>
            <StatusBar siteId={siteId} />
            <Toasts />
            <CommandPalette siteId={siteId} destinations={destinations} />
        </div>
    );
}
