export type Tone = "accent" | "ok" | "warn" | "danger" | "info" | "muted";

export interface ToneClasses {
    readonly ink: string;
    readonly soft: string;
    readonly border: string;
    readonly solid: string;
    readonly onSolid: string;
}

export const toneClasses: Readonly<Record<Tone, ToneClasses>> = {
    accent: {
        ink: "text-accent",
        soft: "bg-accent-soft",
        border: "border-accent-border",
        solid: "bg-accent",
        onSolid: "text-on-accent",
    },
    ok: {
        ink: "text-ok",
        soft: "bg-ok-soft",
        border: "border-ok-border",
        solid: "bg-ok",
        onSolid: "text-on-ok",
    },
    warn: {
        ink: "text-warn",
        soft: "bg-warn-soft",
        border: "border-warn-border",
        solid: "bg-warn",
        onSolid: "text-on-warn",
    },
    danger: {
        ink: "text-danger",
        soft: "bg-danger-soft",
        border: "border-danger-border",
        solid: "bg-danger",
        onSolid: "text-on-danger",
    },
    info: {
        ink: "text-info",
        soft: "bg-info-soft",
        border: "border-info-border",
        solid: "bg-info",
        onSolid: "text-on-info",
    },
    muted: {
        ink: "text-muted",
        soft: "bg-muted-soft",
        border: "border-muted-border",
        solid: "bg-muted",
        onSolid: "text-on-muted",
    },
};
