import type { ReactElement } from "react";

import { Button, Input } from "../../../ui/index.js";

export interface ThresholdProps {
    label: string;
    action: string;
    value: string;
    matched: number;
    onChange: (value: string) => void;
    onApply: () => void;
}

export function Threshold({ label, action, value, matched, onChange, onApply }: ThresholdProps): ReactElement {
    return (
        <span className="flex items-center gap-1.5">
            <span className="text-2xs whitespace-nowrap text-ink-dim">{label}</span>
            <div className="w-16 shrink-0">
                <Input
                    type="number"
                    size="sm"
                    min={0}
                    max={1}
                    step={0.05}
                    mono={true}
                    aria-label={label}
                    value={value}
                    onChange={(event) => {
                        onChange(event.target.value);
                    }}
                />
            </div>
            <Button size="sm" variant="secondary" disabled={matched === 0} onClick={onApply}>
                {action}
            </Button>
        </span>
    );
}
