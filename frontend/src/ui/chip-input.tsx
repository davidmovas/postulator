import type { KeyboardEvent, ReactElement } from "react";
import { useState } from "react";

import { cx } from "./cx.js";
import { CloseIcon } from "./icons/index.js";

export interface ChipInputProps {
    id?: string;
    values: readonly string[];
    onChange: (values: string[]) => void;
    placeholder?: string;
    removeLabel: string;
    mono?: boolean;
    disabled?: boolean;
    invalid?: boolean;
    "aria-describedby"?: string;
}

export function ChipInput({
    id,
    values,
    onChange,
    placeholder,
    removeLabel,
    mono = false,
    disabled = false,
    invalid = false,
    "aria-describedby": describedBy,
}: ChipInputProps): ReactElement {
    const [draft, setDraft] = useState("");

    const commit = (): void => {
        const next = draft.trim();
        if (next === "") {
            return;
        }
        if (!values.some((held) => held.toLowerCase() === next.toLowerCase())) {
            onChange([...values, next]);
        }
        setDraft("");
    };

    const onKeyDown = (event: KeyboardEvent<HTMLInputElement>): void => {
        if (event.key === "Enter" || event.key === ",") {
            event.preventDefault();
            commit();
        } else if (event.key === "Backspace" && draft === "" && values.length > 0) {
            onChange(values.slice(0, -1));
        }
    };

    return (
        <div
            className={cx(
                "flex min-h-7 w-full flex-wrap items-center gap-1 rounded-md border bg-inset px-1.5 py-1",
                invalid ? "border-danger" : "border-hairline focus-within:border-edge",
                disabled && "cursor-not-allowed opacity-70",
            )}
        >
            {values.map((value) => (
                <span
                    key={value}
                    className={cx("inline-flex h-5 items-center gap-1 rounded-sm bg-raised pr-0.5 pl-1.5 text-xs text-ink", mono && "font-mono")}
                >
                    {value}
                    <button
                        type="button"
                        aria-label={`${removeLabel} ${value}`}
                        disabled={disabled}
                        className="flex h-4 w-4 items-center justify-center rounded-sm text-ink-dim hover:bg-raised-strong hover:text-ink"
                        onClick={() => {
                            onChange(values.filter((held) => held !== value));
                        }}
                    >
                        <CloseIcon size={12} />
                    </button>
                </span>
            ))}
            <input
                id={id}
                type="text"
                value={draft}
                placeholder={values.length === 0 ? placeholder : undefined}
                disabled={disabled}
                aria-invalid={invalid || undefined}
                aria-describedby={describedBy}
                className={cx("h-5 min-w-16 flex-1 bg-transparent text-xs text-ink outline-none placeholder:text-ink-faint", mono && "font-mono")}
                onChange={(event) => {
                    setDraft(event.target.value);
                }}
                onKeyDown={onKeyDown}
                onBlur={commit}
            />
        </div>
    );
}
