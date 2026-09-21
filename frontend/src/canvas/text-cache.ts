export interface TextMeasurer {
    font: string;
    measureText(text: string): { width: number };
}

const ellipsis = "…";

export class TextCache {
    private readonly widths = new Map<string, number>();

    constructor(private readonly measurer: TextMeasurer) {}

    clear(): void {
        this.widths.clear();
    }

    width(font: string, text: string): number {
        const key = `${font}\u0000${text}`;
        const held = this.widths.get(key);
        if (held !== undefined) {
            return held;
        }
        if (this.measurer.font !== font) {
            this.measurer.font = font;
        }
        const measured = this.measurer.measureText(text).width;
        this.widths.set(key, measured);
        return measured;
    }

    ellipsise(font: string, text: string, maxWidth: number): string {
        if (this.width(font, text) <= maxWidth) {
            return text;
        }
        const glyphs = [...text];
        let low = 0;
        let high = glyphs.length;
        while (low < high) {
            const middle = Math.ceil((low + high) / 2);
            if (this.width(font, glyphs.slice(0, middle).join("") + ellipsis) <= maxWidth) {
                low = middle;
            } else {
                high = middle - 1;
            }
        }
        if (low === 0) {
            return this.width(font, ellipsis) <= maxWidth ? ellipsis : "";
        }
        return glyphs.slice(0, low).join("") + ellipsis;
    }
}
