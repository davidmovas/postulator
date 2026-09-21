import type { KeyboardEvent, ReactElement, ReactNode } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { Button, IconButton, SendIcon, StopCircleIcon } from "../../../ui/index.js";

const maxComposerHeightPx = 168;

export interface ComposerProps {
    answering: boolean;
    stopping: boolean;
    disabled: boolean;
    placeholder: string;
    error: string | null;
    status: ReactNode;
    prefillSeq: number;
    takePrefill: () => string | null;
    onSend: (text: string) => void;
    onStop: () => void;
}

export function Composer({
    answering,
    stopping,
    disabled,
    placeholder,
    error,
    status,
    prefillSeq,
    takePrefill,
    onSend,
    onStop,
}: ComposerProps): ReactElement {
    const field = useRef<HTMLTextAreaElement>(null);
    const [draft, setDraft] = useState("");

    useLayoutEffect(() => {
        const held = field.current;
        if (held === null) {
            return;
        }
        held.style.height = "auto";
        held.style.height = `${Math.min(held.scrollHeight, maxComposerHeightPx)}px`;
    }, [draft]);

    useEffect(() => {
        if (prefillSeq === 0) {
            return;
        }
        const held = takePrefill();
        if (held !== null) {
            setDraft((current) => (current === "" ? held : `${held}${current}`));
        }
        field.current?.focus();
    }, [prefillSeq, takePrefill]);

    const ready = draft.trim() !== "" && !answering && !disabled;

    const submit = (): void => {
        if (!ready) {
            return;
        }
        onSend(draft.trim());
        setDraft("");
    };

    const keyed = (event: KeyboardEvent<HTMLTextAreaElement>): void => {
        if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            submit();
        }
    };

    return (
        <div className="flex shrink-0 flex-col gap-2 border-t border-hairline px-3 pt-2 pb-2">
            {status}
            <div className="flex items-end gap-2 rounded-lg border border-hairline bg-inset px-2.5 py-2 focus-within:border-edge">
                <textarea
                    ref={field}
                    rows={1}
                    value={draft}
                    disabled={disabled}
                    placeholder={placeholder}
                    aria-label={copy.agent.send}
                    onChange={(event) => {
                        setDraft(event.target.value);
                    }}
                    onKeyDown={keyed}
                    className="min-h-6 w-full resize-none bg-transparent text-sm leading-relaxed text-ink outline-none placeholder:text-ink-faint disabled:cursor-not-allowed"
                />
                {answering ? (
                    <Button
                        size="sm"
                        variant="secondary"
                        icon={StopCircleIcon}
                        busy={stopping}
                        disabled={stopping}
                        onClick={onStop}
                    >
                        {stopping ? copy.agent.status.stopping : copy.agent.cancelTurn}
                    </Button>
                ) : (
                    <IconButton
                        icon={SendIcon}
                        label={copy.agent.send}
                        variant="primary"
                        size="sm"
                        disabled={!ready}
                        onClick={submit}
                    />
                )}
            </div>
            {error === null ? null : <span className="truncate px-0.5 text-2xs text-danger">{error}</span>}
        </div>
    );
}
