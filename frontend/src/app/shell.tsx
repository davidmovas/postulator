import { PanelRightClose, PanelRightOpen, X } from "lucide-react";
import { useState } from "react";
import { Outlet, useLocation } from "react-router";

import { copy } from "../copy/index.js";
import { dismissToast, useToasts } from "../data/toasts.js";
import { NotBuilt } from "./not-built.js";
import { Rail } from "./rail.js";
import { StatusBar } from "./statusbar.js";
import { TitleBar } from "./titlebar.js";

const dockWidthKey = "postulator.dock.width";
const minimumDockWidth = 280;
const maximumDockWidth = 640;

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

function ToastRegion() {
    const toasts = useToasts();
    if (toasts.length === 0) {
        return null;
    }
    return (
        <div className="pointer-events-none fixed bottom-8 right-4 z-50 flex w-80 flex-col gap-2">
            {toasts.map((toast) => (
                <div
                    key={toast.id}
                    className={`pointer-events-auto flex items-start gap-2 rounded-panel border px-3 py-2 text-xs shadow-lg ${
                        toast.tone === "danger"
                            ? "border-bad-500 bg-base-800 text-ink-100"
                            : toast.tone === "warning"
                              ? "border-warn-500 bg-base-800 text-ink-100"
                              : "border-base-600 bg-base-800 text-ink-200"
                    }`}
                >
                    <span className="flex-1">{toast.message}</span>
                    <button
                        type="button"
                        aria-label={copy.app.dismiss}
                        className="text-ink-400 hover:text-ink-100"
                        onClick={() => {
                            dismissToast(toast.id);
                        }}
                    >
                        <X size={13} />
                    </button>
                </div>
            ))}
        </div>
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
        <div className="flex h-full flex-col">
            <TitleBar siteId={siteId} />
            <div className="flex min-h-0 flex-1">
                <Rail siteId={siteId} />
                <main className="min-w-0 flex-1 overflow-auto">
                    <Outlet />
                </main>
                {dockOpen ? (
                    <aside
                        className="flex shrink-0 flex-col border-l border-base-700 bg-base-900"
                        style={{ width: `${dockWidth}px` }}
                    >
                        <div className="flex h-8 items-center justify-between border-b border-base-700 px-2">
                            <span className="text-xs text-ink-300">{copy.shell.agentDock}</span>
                            <div className="flex items-center gap-1">
                                <button
                                    type="button"
                                    className="text-ink-400 hover:text-ink-100"
                                    aria-label={copy.shell.collapseDock}
                                    onClick={() => {
                                        setDockOpen(false);
                                    }}
                                >
                                    <PanelRightClose size={15} />
                                </button>
                            </div>
                        </div>
                        <input
                            type="range"
                            aria-label={copy.shell.agentDock}
                            className="mx-2 mt-2 accent-accent-500"
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
                    <button
                        type="button"
                        className="w-8 shrink-0 border-l border-base-700 bg-base-900 text-ink-400 hover:text-ink-100"
                        aria-label={copy.shell.expandDock}
                        onClick={() => {
                            setDockOpen(true);
                        }}
                    >
                        <PanelRightOpen size={15} className="mx-auto" />
                    </button>
                )}
            </div>
            <StatusBar siteId={siteId} />
            <ToastRegion />
        </div>
    );
}
