import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";

export interface WindowControlsProps {
    maximised: boolean;
    minimiseLabel: string;
    maximiseLabel: string;
    restoreLabel: string;
    closeLabel: string;
    onMinimise: () => void;
    onToggleMaximise: () => void;
    onClose: () => void;
}

interface ControlProps {
    label: string;
    danger?: boolean;
    onClick: () => void;
    children: ReactNode;
}

function Control({ label, danger = false, onClick, children }: ControlProps): ReactElement {
    return (
        <button
            type="button"
            aria-label={label}
            title={label}
            onClick={onClick}
            className={cx(
                "inline-flex h-10 w-[46px] shrink-0 items-center justify-center text-ink-dim",
                "transition-colors duration-100 ease-out",
                danger ? "hover:bg-danger hover:text-on-danger" : "hover:bg-inset hover:text-ink",
            )}
        >
            {children}
        </button>
    );
}

function Glyph({ children }: { children: ReactNode }): ReactElement {
    return (
        <svg
            width={10}
            height={10}
            viewBox="0 0 10 10"
            aria-hidden={true}
            fill="none"
            stroke="currentColor"
            strokeWidth={1}
            strokeLinecap="square"
        >
            {children}
        </svg>
    );
}

export function WindowControls({
    maximised,
    minimiseLabel,
    maximiseLabel,
    restoreLabel,
    closeLabel,
    onMinimise,
    onToggleMaximise,
    onClose,
}: WindowControlsProps): ReactElement {
    return (
        <div className="flex shrink-0 items-center">
            <Control label={minimiseLabel} onClick={onMinimise}>
                <Glyph>
                    <path d="M0.5 5.5 H9.5" />
                </Glyph>
            </Control>
            <Control label={maximised ? restoreLabel : maximiseLabel} onClick={onToggleMaximise}>
                <Glyph>
                    {maximised ? (
                        <>
                            <path d="M0.5 2.5 H7.5 V9.5 H0.5 Z" />
                            <path d="M2.5 2.5 V0.5 H9.5 V7.5 H7.5" />
                        </>
                    ) : (
                        <path d="M0.5 0.5 H9.5 V9.5 H0.5 Z" />
                    )}
                </Glyph>
            </Control>
            <Control label={closeLabel} danger={true} onClick={onClose}>
                <Glyph>
                    <path d="M0.5 0.5 L9.5 9.5" />
                    <path d="M9.5 0.5 L0.5 9.5" />
                </Glyph>
            </Control>
        </div>
    );
}
