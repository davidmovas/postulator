export type CardKeys = "confirm" | "list" | "none";

export type CardChoice = "approve" | "reject" | null;

export interface CardStroke {
    key: string;
    ctrlKey: boolean;
    metaKey: boolean;
}

export function cardChoice(keys: CardKeys, stroke: CardStroke): CardChoice {
    if (keys !== "confirm") {
        return null;
    }
    return stroke.key === "Enter" && (stroke.ctrlKey || stroke.metaKey) ? "approve" : null;
}
