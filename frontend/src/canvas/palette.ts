export interface Palette {
    canvas: string;
    panel: string;
    inset: string;
    raised: string;
    raisedStrong: string;
    hairline: string;
    edge: string;
    ink: string;
    inkSoft: string;
    inkDim: string;
    inkFaint: string;
    accent: string;
    accentSoft: string;
    accentBorder: string;
    ok: string;
    okSoft: string;
    warn: string;
    warnSoft: string;
    danger: string;
    dangerSoft: string;
    info: string;
    infoSoft: string;
    muted: string;
    mutedSoft: string;
    fontSans: string;
    fontMono: string;
}

const colorTokens: Readonly<Record<Exclude<keyof Palette, "fontSans" | "fontMono">, string>> = {
    canvas: "--color-canvas",
    panel: "--color-panel",
    inset: "--color-inset",
    raised: "--color-raised",
    raisedStrong: "--color-raised-strong",
    hairline: "--color-hairline",
    edge: "--color-edge",
    ink: "--color-ink",
    inkSoft: "--color-ink-soft",
    inkDim: "--color-ink-dim",
    inkFaint: "--color-ink-faint",
    accent: "--color-accent",
    accentSoft: "--color-accent-soft",
    accentBorder: "--color-accent-border",
    ok: "--color-ok",
    okSoft: "--color-ok-soft",
    warn: "--color-warn",
    warnSoft: "--color-warn-soft",
    danger: "--color-danger",
    dangerSoft: "--color-danger-soft",
    info: "--color-info",
    infoSoft: "--color-info-soft",
    muted: "--color-muted",
    mutedSoft: "--color-muted-soft",
};

function resolveColor(probe: HTMLElement, raw: string): string {
    probe.style.color = raw;
    return getComputedStyle(probe).color;
}

export function readPalette(element: HTMLElement): Palette {
    const computed = getComputedStyle(element);
    const probe = element.ownerDocument.createElement("span");
    probe.style.position = "absolute";
    probe.style.visibility = "hidden";
    element.appendChild(probe);
    try {
        const palette: Record<string, string> = {};
        for (const [name, token] of Object.entries(colorTokens)) {
            palette[name] = resolveColor(probe, computed.getPropertyValue(token).trim());
        }
        palette.fontSans = computed.getPropertyValue("--font-sans").trim();
        palette.fontMono = computed.getPropertyValue("--font-mono").trim();
        return palette as unknown as Palette;
    } finally {
        probe.remove();
    }
}
