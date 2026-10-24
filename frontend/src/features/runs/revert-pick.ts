import type { PublishView } from "./artifacts.js";

const pageBody = "body";
const productBody = "product_description";

export function bodyKeyOf(view: PublishView): string {
    return view.product === null ? pageBody : productBody;
}

export function bodyWritten(view: PublishView): boolean {
    return view.blocks.some((block) => block.change !== "kept");
}

export function revertableKeys(view: PublishView): string[] {
    if (view.created) {
        return [];
    }
    const fields = view.fields.filter((change) => change.beforeHash !== change.afterHash).map((change) => change.key);
    const body = bodyWritten(view) ? [bodyKeyOf(view)] : [];
    return [...new Set([...fields, ...body])];
}

export function picked(offered: readonly string[], ticked: ReadonlySet<string>): string[] {
    return offered.filter((key) => ticked.has(key));
}
