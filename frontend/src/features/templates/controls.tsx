import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { react } from "../../data/errors.js";
import { Input } from "../../ui/index.js";

export function fieldErrorOf(thrown: unknown, field: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "field" && reaction.field === field ? reaction.message : null;
}

export function fieldErrorUnder(thrown: unknown, prefix: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    if (reaction.kind !== "field") {
        return null;
    }
    return reaction.field === prefix || reaction.field.startsWith(`${prefix}.`) ? reaction.message : null;
}

export function formErrorOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "form" ? reaction.message : null;
}

export interface NumberInputProps {
    value: number;
    onValueChange: (value: number) => void;
    id?: string;
    describedBy?: string;
    invalid?: boolean;
    min?: number;
    max?: number;
    step?: number;
    className?: string;
}

export function NumberInput({
    value,
    onValueChange,
    id,
    describedBy,
    invalid = false,
    min,
    max,
    step,
    className,
}: NumberInputProps): ReactElement {
    const [text, setText] = useState(() => String(value));

    useEffect(() => {
        setText((held) => (Number(held) === value ? held : String(value)));
    }, [value]);

    return (
        <Input
            id={id}
            aria-describedby={describedBy}
            invalid={invalid}
            mono={true}
            type="number"
            min={min}
            max={max}
            step={step}
            className={className}
            value={text}
            onChange={(event) => {
                const raw = event.target.value;
                setText(raw);
                const parsed = Number(raw);
                if (raw !== "" && Number.isFinite(parsed)) {
                    onValueChange(parsed);
                }
            }}
        />
    );
}
