export interface Track {
    track: string;
    least: number;
}

export type Tracks<C extends string> = Readonly<Record<C, Track>>;

export interface ColumnLayout<C extends string> {
    columns: readonly C[];
    template: string;
    least: number;
}

const columnGap = 12;

export function laidOut<C extends string>(columns: readonly C[], tracks: Tracks<C>, chrome: number): ColumnLayout<C> {
    return {
        columns,
        template: columns.map((column) => tracks[column].track).join(" "),
        least: columns.reduce((sum, column) => sum + tracks[column].least, chrome + columnGap * (columns.length - 1)),
    };
}

export function widestFitting<C extends string>(
    width: number,
    layouts: readonly ColumnLayout<C>[],
    narrowest: ColumnLayout<C>,
): ColumnLayout<C> {
    if (width <= 0) {
        return layouts[0] ?? narrowest;
    }
    return layouts.find((layout) => layout.least <= width) ?? narrowest;
}

export function shown<C extends string>(layout: ColumnLayout<C>, column: C): boolean {
    return layout.columns.includes(column);
}
