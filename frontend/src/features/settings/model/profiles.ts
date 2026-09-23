export type ProfileSource = "global" | "seeded" | "none";

export interface ModelRefLike {
    provider: string;
    model: string;
}

export interface ProfileLike {
    role: string;
    global?: ModelRefLike | null;
    effective?: ModelRefLike | null;
}

export interface ProfileRow {
    role: string;
    chosen: ModelRefLike | null;
    effective: ModelRefLike | null;
    source: ProfileSource;
}

export const refSeparator = "/";

export function refText(ref: ModelRefLike | null): string {
    return ref === null ? "" : `${ref.provider}${refSeparator}${ref.model}`;
}

export function refOf(text: string): ModelRefLike | null {
    const cut = text.indexOf(refSeparator);
    if (cut <= 0 || cut === text.length - 1) {
        return null;
    }
    return { provider: text.slice(0, cut), model: text.slice(cut + 1) };
}

function usable(ref: ModelRefLike | null | undefined): ModelRefLike | null {
    if (ref === null || ref === undefined) {
        return null;
    }
    return ref.provider === "" || ref.model === "" ? null : ref;
}

export function profileRows(roles: readonly string[], profiles: readonly ProfileLike[]): ProfileRow[] {
    const byRole = new Map<string, ProfileLike>();
    for (const profile of profiles) {
        byRole.set(profile.role, profile);
    }

    return roles.map((role) => {
        const held = byRole.get(role);
        const chosen = usable(held?.global);
        const effective = usable(held?.effective);
        if (chosen !== null) {
            return { role, chosen, effective, source: "global" };
        }
        return { role, chosen: null, effective, source: effective === null ? "none" : "seeded" };
    });
}
