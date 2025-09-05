// shadcn/ui toast implementation (TS) with increased limit
// Source adapted from shadcn/ui (MIT)

import * as React from "react";

export type Toast = {
  id?: string;
  title?: string;
  description?: React.ReactNode;
  action?: React.ReactNode;
  dismissible?: boolean;
  duration?: number;
  variant?: "default" | "destructive" | "success" | "warning" | "info";
};

export type ToasterToast = Toast & {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

// Increase the toast limit to allow multiple to show at the same time
export const TOAST_LIMIT = 6;
export const TOAST_REMOVE_DELAY = 1000;

const actionTypes = {
  ADD_TOAST: "ADD_TOAST",
  UPDATE_TOAST: "UPDATE_TOAST",
  DISMISS_TOAST: "DISMISS_TOAST",
  REMOVE_TOAST: "REMOVE_TOAST",
} as const;

type ActionType = typeof actionTypes;

type State = {
  toasts: ToasterToast[];
};

let count = 0;
function genId() {
  count = (count + 1) % Number.MAX_SAFE_INTEGER;
  return count.toString();
}

// Timeout map for delayed removal after dismiss (fade-out)
const toastRemoveTimeouts = new Map<string, ReturnType<typeof setTimeout>>();
// Timeout map for auto-dismiss after duration
const toastAutoDismissTimers = new Map<string, ReturnType<typeof setTimeout>>();

const addToRemoveQueue = (toastId: string) => {
  if (toastRemoveTimeouts.has(toastId)) return;

  const timeout = setTimeout(() => {
    toastRemoveTimeouts.delete(toastId);
    dispatch({ type: "REMOVE_TOAST", toastId });
  }, TOAST_REMOVE_DELAY);

  toastRemoveTimeouts.set(toastId, timeout);
};

const startAutoDismissTimer = (t: ToasterToast) => {
  const id = t.id as string;
  // Clear any existing timer first
  const prev = toastAutoDismissTimers.get(id);
  if (prev) clearTimeout(prev);

  const duration = typeof t.duration === "number" ? t.duration : 4000;
  // Do not auto-dismiss if duration is Infinity or 0/negative
  if (!isFinite(duration) || duration <= 0) return;

  const timer = setTimeout(() => {
    // Auto-dismiss triggers a normal dismiss (which schedules removal after delay)
    dispatch({ type: "DISMISS_TOAST", toastId: id });
    toastAutoDismissTimers.delete(id);
  }, duration);

  toastAutoDismissTimers.set(id, timer);
};

const clearAutoDismissTimer = (id?: string) => {
  if (!id) return;
  const timer = toastAutoDismissTimers.get(id);
  if (timer) {
    clearTimeout(timer);
    toastAutoDismissTimers.delete(id);
  }
};

const defaultToast: Partial<Toast> = {
  dismissible: true,
  duration: 4000,
  variant: "default",
};

const listeners: Array<(state: State) => void> = [];
let memoryState: State = { toasts: [] };

function dispatch(action: { type: ActionType[keyof ActionType]; toast?: Toast; toastId?: string }) {
  switch (action.type) {
    case "ADD_TOAST": {
      const toast = {
        ...defaultToast,
        ...action.toast,
        id: action.toast?.id ?? genId(),
        open: true,
        onOpenChange: (open: boolean) => {
          if (!open) dispatch({ type: "DISMISS_TOAST", toastId: toast.id });
        },
      } as ToasterToast;

      memoryState = {
        ...memoryState,
        toasts: [toast, ...memoryState.toasts].slice(0, TOAST_LIMIT),
      };

      // Start auto-dismiss timer for this toast
      startAutoDismissTimer(toast);
      break;
    }
    case "UPDATE_TOAST": {
      // Apply update
      memoryState = {
        ...memoryState,
        toasts: memoryState.toasts.map((t) => (t.id === action.toast?.id ? ({ ...t, ...action.toast } as ToasterToast) : t)),
      };
      // Restart timer if duration changed or ensure timer exists
      if (action.toast?.id) {
        const updated = memoryState.toasts.find((t) => t.id === action.toast?.id);
        if (updated) startAutoDismissTimer(updated);
      }
      break;
    }
    case "DISMISS_TOAST": {
      const { toastId } = action;
      // Stop auto-dismiss timer when we dismiss
      if (toastId) clearAutoDismissTimer(toastId);
      // Dismiss a specific toast
      if (toastId) addToRemoveQueue(toastId);
      // Or dismiss all toasts
      memoryState = {
        ...memoryState,
        toasts: memoryState.toasts.map((t) => (t.id === toastId || toastId === undefined ? { ...t, open: false } : t)),
      };
      break;
    }
    case "REMOVE_TOAST": {
      // Ensure all timers related to this toast are cleared
      if (action.toastId) {
        const removeTimeout = toastRemoveTimeouts.get(action.toastId);
        if (removeTimeout) {
          clearTimeout(removeTimeout);
          toastRemoveTimeouts.delete(action.toastId);
        }
        clearAutoDismissTimer(action.toastId);
      }
      memoryState = {
        ...memoryState,
        toasts: memoryState.toasts.filter((t) => t.id !== action.toastId),
      };
      break;
    }
  }

  listeners.forEach((listener) => {
    listener(memoryState);
  });
}

export function useToast() {
  const [state, setState] = React.useState<State>(memoryState);

  React.useEffect(() => {
    listeners.push(setState);
    return () => {
      const index = listeners.indexOf(setState);
      if (index > -1) listeners.splice(index, 1);
    };
  }, [state]);

  const toast = React.useCallback((toast: Toast) => {
    dispatch({ type: "ADD_TOAST", toast });
    return toast.id ?? "";
  }, []);

  const dismiss = React.useCallback((toastId?: string) => {
    if (toastId) {
      const timeout = toastRemoveTimeouts.get(toastId);
      if (timeout) {
        clearTimeout(timeout);
        toastRemoveTimeouts.delete(toastId);
      }
      // Immediate removal on explicit dismiss (close button)
      dispatch({ type: "REMOVE_TOAST", toastId });
    } else {
      // Bulk dismiss (no id): keep current behavior (close, then remove by queue if elsewhere scheduled)
      dispatch({ type: "DISMISS_TOAST" });
    }
  }, []);

  return {
    ...state,
    toast,
    dismiss,
  };
}

export type { State as ToastState };
