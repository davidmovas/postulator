import type { InputHTMLAttributes, ReactElement, ReactNode, TextareaHTMLAttributes } from "react";
import { useId } from "react";

import { cx } from "./cx.js";
import { ErrorIcon } from "./icons/index.js";

const controlBase =
    "w-full rounded-md bg-inset text-ink placeholder:text-ink-faint transition-colors duration-100 ease-out disabled:cursor-not-allowed disabled:opacity-70";

export interface ControlBinding {
    id: string;
    "aria-describedby": string | undefined;
    invalid: boolean;
}

export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "size"> {
    invalid?: boolean;
    mono?: boolean;
}

export function Input({ invalid = false, mono = false, className, type = "text", ...rest }: InputProps): ReactElement {
    return (
        <input
            type={type}
            aria-invalid={invalid || undefined}
            className={cx(
                controlBase,
                "h-7 border px-2.5 text-sm",
                mono && "font-mono",
                invalid ? "border-danger" : "border-hairline focus:border-edge",
                className,
            )}
            {...rest}
        />
    );
}

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
    invalid?: boolean;
    mono?: boolean;
}

export function Textarea({ invalid = false, mono = false, className, rows = 4, ...rest }: TextareaProps): ReactElement {
    return (
        <textarea
            rows={rows}
            aria-invalid={invalid || undefined}
            className={cx(
                controlBase,
                "resize-y border px-2.5 py-1.5 text-sm",
                mono && "font-mono",
                invalid ? "border-danger" : "border-hairline focus:border-edge",
                className,
            )}
            {...rest}
        />
    );
}

export interface FieldProps {
    label: string;
    children: (control: ControlBinding) => ReactNode;
    hint?: string;
    tooltip?: string;
    error?: string | null;
    required?: boolean;
    className?: string;
}

export function Field({ label, children, hint, tooltip, error, required = false, className }: FieldProps): ReactElement {
    const base = useId();
    const controlId = `${base}-control`;
    const noteId = `${base}-note`;
    const failed = error !== undefined && error !== null && error.length > 0;
    const note = failed ? error : (hint ?? null);

    return (
        <div className={cx("flex flex-col gap-1", className)}>
            <label htmlFor={controlId} title={tooltip} className="w-fit text-xs font-medium text-ink-soft">
                {label}
                {required ? <span className="ml-0.5 text-danger">*</span> : null}
            </label>
            {children({
                id: controlId,
                "aria-describedby": note === null ? undefined : noteId,
                invalid: failed,
            })}
            {note === null ? null : (
                <p
                    id={noteId}
                    className={cx("flex items-start gap-1 text-xs", failed ? "text-danger" : "text-ink-dim")}
                >
                    {failed ? <ErrorIcon size={14} className="mt-px shrink-0" /> : null}
                    {note}
                </p>
            )}
        </div>
    );
}
