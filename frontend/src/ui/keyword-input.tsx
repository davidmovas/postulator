import type { KeyboardEvent, ReactElement } from "react";
import { useState } from "react";

import type { Keyword } from "../data/types.js";
import { keywordList, volumeLabel } from "../domain/keywords.js";
import { cx } from "./cx.js";
import { CloseIcon } from "./icons/index.js";

export interface KeywordInputProps {
    id?: string;
    values: readonly Keyword[];
    onChange: (values: Keyword[]) => void;
    removeLabel: string;
    phraseLabel: string;
    volumeLabel: string;
    disabled?: boolean;
    invalid?: boolean;
    "aria-describedby"?: string;
}

function volumeOf(raw: string): number | null {
    const trimmed = raw.trim();
    if (!/^\d+$/.test(trimmed)) {
        return null;
    }
    return Number(trimmed);
}

export function KeywordInput({
    id,
    values,
    onChange,
    removeLabel,
    phraseLabel,
    volumeLabel: volumeName,
    disabled = false,
    invalid = false,
    "aria-describedby": describedBy,
}: KeywordInputProps): ReactElement {
    const [phrase, setPhrase] = useState("");
    const [volume, setVolume] = useState("");
    const shown = keywordList(values);

    const commit = (): void => {
        const text = phrase.trim();
        if (text === "") {
            return;
        }
        const measured = volumeOf(volume);
        const typed: Keyword = measured === null ? { text } : { text, volume: measured };
        const held = shown.findIndex((item) => item.text.toLowerCase() === text.toLowerCase());
        const next = held < 0 ? [...shown, typed] : shown.map((item, at) => (at === held ? { ...item, ...(measured === null ? {} : { volume: measured }) } : item));
        onChange(keywordList(next));
        setPhrase("");
        setVolume("");
    };

    const onKeyDown = (event: KeyboardEvent<HTMLInputElement>): void => {
        if (event.key === "Enter") {
            event.preventDefault();
            commit();
        }
    };

    return (
        <div
            className={cx(
                "flex w-full flex-col gap-1 rounded-md border bg-inset px-1.5 py-1",
                invalid ? "border-danger" : "border-hairline focus-within:border-edge",
                disabled && "cursor-not-allowed opacity-70",
            )}
        >
            {shown.length === 0 ? null : (
                <ul className="flex flex-wrap gap-1">
                    {shown.map((item) => (
                        <li key={item.text} className="inline-flex h-5 items-center gap-1 rounded-sm bg-raised pr-0.5 pl-1.5 font-mono text-xs text-ink">
                            <span>{item.text}</span>
                            {volumeLabel(item) === "" ? null : <span className="text-2xs text-ink-faint tabular-nums">{volumeLabel(item)}</span>}
                            <button
                                type="button"
                                aria-label={`${removeLabel} ${item.text}`}
                                disabled={disabled}
                                className="flex h-4 w-4 items-center justify-center rounded-sm text-ink-dim hover:bg-raised-strong hover:text-ink"
                                onClick={() => {
                                    onChange(shown.filter((held) => held.text !== item.text));
                                }}
                            >
                                <CloseIcon size={12} />
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            <div className="flex items-center gap-1">
                <input
                    id={id}
                    type="text"
                    value={phrase}
                    disabled={disabled}
                    aria-label={phraseLabel}
                    aria-invalid={invalid || undefined}
                    aria-describedby={describedBy}
                    placeholder={phraseLabel}
                    className="h-5 min-w-16 flex-1 bg-transparent font-mono text-xs text-ink outline-none placeholder:text-ink-faint"
                    onChange={(event) => {
                        setPhrase(event.target.value);
                    }}
                    onKeyDown={onKeyDown}
                />
                <input
                    type="text"
                    inputMode="numeric"
                    value={volume}
                    disabled={disabled}
                    aria-label={volumeName}
                    placeholder={volumeName}
                    className="h-5 w-20 bg-transparent text-right text-xs text-ink tabular-nums outline-none placeholder:text-ink-faint"
                    onChange={(event) => {
                        setVolume(event.target.value);
                    }}
                    onKeyDown={onKeyDown}
                    onBlur={commit}
                />
            </div>
        </div>
    );
}
