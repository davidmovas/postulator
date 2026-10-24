export type MapColumn = "page" | "type" | "parent" | "body" | "links" | "findings" | "children" | "under";

export interface MapLayout {
    columns: readonly MapColumn[];
    template: string;
    least: number;
}

interface Track {
    track: string;
    least: number;
}

type Tracks = Readonly<Record<MapColumn, Track>>;

const tracks: Tracks = {
    page: { track: "minmax(260px,3fr)", least: 260 },
    type: { track: "minmax(72px,1fr)", least: 72 },
    parent: { track: "minmax(120px,1.4fr)", least: 120 },
    body: { track: "76px", least: 76 },
    links: { track: "64px", least: 64 },
    findings: { track: "64px", least: 64 },
    children: { track: "52px", least: 52 },
    under: { track: "28px", least: 28 },
};

const shrinking: Tracks = {
    ...tracks,
    page: { track: "minmax(0,3fr)", least: 0 },
    type: { track: "minmax(0,1fr)", least: 0 },
};

const gap = 12;

const rowChrome = 34;

const everyColumn: readonly MapColumn[] = ["page", "type", "parent", "body", "links", "findings", "children", "under"];

const withParent: readonly MapColumn[] = ["page", "type", "parent", "body", "findings", "under"];

const keyFacts: readonly MapColumn[] = ["page", "type", "body", "findings", "under"];

function laid(columns: readonly MapColumn[], held: Tracks): MapLayout {
    return {
        columns,
        template: columns.map((column) => held[column].track).join(" "),
        least: columns.reduce((sum, column) => sum + held[column].least, rowChrome + gap * (columns.length - 1)),
    };
}

export function mapLayout(width: number): MapLayout {
    const every = laid(everyColumn, tracks);
    if (width <= 0) {
        return every;
    }
    return [every, laid(withParent, tracks)].find((layout) => layout.least <= width) ?? laid(keyFacts, shrinking);
}
