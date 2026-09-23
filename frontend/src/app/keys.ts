export interface Stroke {
    key: string;
    ctrl: boolean;
    meta: boolean;
    alt: boolean;
}

export interface Where {
    editable: boolean;
    overlayOpen: boolean;
}

export function opensPalette(stroke: Stroke, where: Where): boolean {
    if (stroke.alt) {
        return false;
    }
    if (stroke.ctrl || stroke.meta) {
        return stroke.key.toLowerCase() === "k";
    }
    return stroke.key.toLowerCase() === "f" && !where.editable && !where.overlayOpen;
}

export function togglesDock(stroke: Stroke): boolean {
    return !stroke.alt && (stroke.ctrl || stroke.meta) && stroke.key.toLowerCase() === "j";
}

const editableTags = new Set(["input", "textarea", "select"]);

export function editableTarget(element: Element | null): boolean {
    if (element === null) {
        return false;
    }
    if (editableTags.has(element.tagName.toLowerCase())) {
        return true;
    }
    return element instanceof HTMLElement && element.isContentEditable;
}
