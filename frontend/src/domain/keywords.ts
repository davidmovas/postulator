import type { Keyword } from "../data/types.js";

const counted = new Intl.NumberFormat("en", { maximumFractionDigits: 0 });

function measured(volume: number | null | undefined): number | null {
    return typeof volume === "number" && Number.isFinite(volume) && volume >= 0 ? volume : null;
}

function byVolume(left: Keyword, right: Keyword): number {
    const a = measured(left.volume);
    const b = measured(right.volume);
    if (a === null && b === null) {
        return 0;
    }
    if (a === null) {
        return 1;
    }
    if (b === null) {
        return -1;
    }
    return b - a;
}

export function keywordList(items: readonly Keyword[] | null | undefined): Keyword[] {
    const out: Keyword[] = [];
    const held = new Map<string, number>();
    for (const item of items ?? []) {
        const text = item.text.trim();
        if (text === "") {
            continue;
        }
        const volume = measured(item.volume);
        const key = text.toLowerCase();
        const at = held.get(key);
        if (at !== undefined) {
            const first = out[at];
            if (first !== undefined && first.volume === undefined && volume !== null) {
                out[at] = { text: first.text, volume };
            }
            continue;
        }
        held.set(key, out.length);
        out.push(volume === null ? { text } : { text, volume });
    }
    return out.sort(byVolume);
}

export function mainKeyword(list: readonly Keyword[]): string {
    return list[0]?.text ?? "";
}

export function keywordTexts(list: readonly Keyword[]): string[] {
    return list.map((item) => item.text);
}

export function sameKeywords(left: readonly Keyword[], right: readonly Keyword[]): boolean {
    return (
        left.length === right.length &&
        left.every((item, index) => {
            const other = right[index];
            return other !== undefined && other.text === item.text && measured(other.volume) === measured(item.volume);
        })
    );
}

export function volumeLabel(item: Keyword): string {
    const volume = measured(item.volume);
    return volume === null ? "" : counted.format(volume);
}

export function keywordsLine(items: readonly Keyword[] | null | undefined): string {
    return keywordList(items)
        .map((item) => (volumeLabel(item) === "" ? item.text : `${item.text} (${volumeLabel(item)})`))
        .join(", ");
}
