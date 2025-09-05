"use client";

import * as React from "react";
import { useToast } from "./use-toast";

export function Toaster() {
  const { toasts, dismiss } = useToast();

  return (
    <div className="fixed inset-0 z-[100] pointer-events-none flex flex-col-reverse gap-2 p-4 sm:flex-col sm:items-end">
      {toasts.map((t) => {
        return (
          <div key={t.id} className="pointer-events-auto w-full sm:max-w-sm">
            <div
              role="status"
              aria-live="polite"
              className={`rounded-md border bg-background text-foreground shadow-lg p-3 text-sm ${
                t.variant === "destructive"
                  ? "border-destructive/50 bg-destructive text-destructive-foreground"
                  : t.variant === "success"
                  ? "border-green-500/40 bg-green-500 text-white"
                  : t.variant === "warning"
                  ? "border-yellow-500/40 bg-yellow-500 text-black"
                  : t.variant === "info"
                  ? "border-blue-500/40 bg-blue-500 text-white"
                  : ""
              }`}
            >
              <div className="flex items-start gap-3">
                <div className="flex-1 min-w-0">
                  {t.title && <div className="font-medium leading-none mb-1 truncate">{t.title}</div>}
                  {t.description && (
                    <div className="text-sm leading-relaxed text-foreground/90 break-words">{t.description}</div>
                  )}
                  {t.action ? <div className="mt-2">{t.action}</div> : null}
                </div>
                {t.dismissible !== false ? (
                  <button
                    aria-label="Dismiss"
                    className="ml-2 text-foreground/70 hover:text-foreground"
                    onClick={() => dismiss(t.id)}
                  >
                    ✕
                  </button>
                ) : null}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default Toaster;
