export type CardKeys = "confirm" | "list" | "none";

export type CardChoice = "approve" | "reject" | null;

export interface CardStroke {
    key: string;
    ctrlKey: boolean;
    metaKey: boolean;
    shiftKey: boolean;
}

export function cardChoice(keys: CardKeys, stroke: CardStroke): CardChoice {
    if (keys !== "confirm" || stroke.key !== "Enter" || !(stroke.ctrlKey || stroke.metaKey)) {
        return null;
    }
    return stroke.shiftKey ? "reject" : "approve";
}
