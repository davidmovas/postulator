export type Detail = "full" | "branches" | "roots";

export const fullDetailZoom = 0.6;
export const branchDetailZoom = 0.35;

export function detailOf(zoom: number): Detail {
    if (zoom >= fullDetailZoom) {
        return "full";
    }
    return zoom >= branchDetailZoom ? "branches" : "roots";
}

export function labelled(detail: Detail, depth: number, childCount: number, selected: boolean): boolean {
    if (selected) {
        return true;
    }
    switch (detail) {
        case "full":
            return true;
        case "branches":
            return childCount > 0;
        case "roots":
            return depth <= 1;
    }
}
