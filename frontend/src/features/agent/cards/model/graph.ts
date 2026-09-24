import { copy } from "../../../../copy/index.js";
import { percent } from "../../../templates/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { line, listed, num, records, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe.graph;

function anchorLines(anchors: readonly Args[]): Line[] {
    return anchors.map((anchor) =>
        line("link", said.anchor(text(anchor, "text") ?? "", percent(num(anchor, "weight") ?? 0), text(anchor, "source") ?? "")),
    );
}

function entityFields(args: Args, renaming: boolean): Line[] {
    const lines: Line[] = [];
    const name = text(args, "name");
    if (renaming && name !== null) {
        lines.push(line("field", said.renamed(name)));
    }
    const kind = text(args, "kind");
    if (kind !== null) {
        lines.push(line("field", said.kind(kind)));
    }
    const intent = text(args, "intent");
    if (intent !== null) {
        lines.push(line("field", said.intent(intent)));
    }
    const primary = text(args, "primaryKeyword");
    if (primary !== null) {
        lines.push(line("target", said.primary(primary)));
    }
    const secondary = strings(args, "secondaryKeywords");
    if (secondary.length > 0) {
        lines.push(line("list", said.secondary(listed(secondary))));
    }
    return lines;
}

export const describeGraph: Describer = (tool, args) => {
    switch (tool) {
        case "graph_create_entity": {
            const anchors = records(args, "anchors");
            const lines = entityFields(args, false);
            if (anchors.length > 0) {
                lines.push(line("list", said.anchorsCount(anchors.length)), ...anchorLines(anchors));
            }
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "graph_create_entities": {
            const entities = records(args, "entities");
            const lines: Line[] = [line("note", said.createManyBody)];
            for (const entity of entities) {
                const name = text(entity, "name") ?? "";
                const parent = text(entity, "parentName");
                lines.push(line(parent === null ? "target" : "link", parent === null ? said.root(name) : said.child(name, parent)));
            }
            return view(said.createMany(entities.length), lines);
        }
        case "graph_update_entity":
            return view([`${said.update} `, ref("entity", text(args, "id") ?? "")], entityFields(args, true));
        case "graph_delete_entity":
            return view([`${said.delete} `, ref("entity", text(args, "id") ?? "")], [line("warn", said.deleteBody, "danger")]);
        case "graph_set_anchors": {
            const anchors = records(args, "anchors");
            return view(
                [`${said.setAnchors} `, ref("entity", text(args, "entityId") ?? "")],
                [line("list", said.anchorsCount(anchors.length)), ...anchorLines(anchors)],
            );
        }
        case "graph_add_edge": {
            const kind = text(args, "kind") ?? "";
            const lines: Line[] = [];
            const weight = num(args, "weight");
            if (weight !== null) {
                lines.push(line("rule", said.weight(percent(weight))));
            }
            const reason = text(args, "reason");
            if (reason !== null) {
                lines.push(line("note", reason));
            }
            return view(
                [
                    `${said.connect} `,
                    ref("entity", text(args, "fromEntityId") ?? ""),
                    ` ${kind === "parent" ? said.under : `${said.with} ${kind}`} `,
                    ref("entity", text(args, "toEntityId") ?? ""),
                ],
                lines,
            );
        }
        case "graph_approve_edge":
            return view([`${said.approve} `, ref("edge", text(args, "id") ?? "")], []);
        case "graph_reject_edge":
            return view([`${said.reject} `, ref("edge", text(args, "id") ?? "")], []);
        case "graph_delete_edge":
            return view([`${said.deleteEdge} `, ref("edge", text(args, "id") ?? "")], []);
        case "graph_recompute_scores":
            return view(said.recompute, [line("note", said.recomputeBody)]);
        case "graph_propose_from_pages":
            return view(said.proposeFromPages, [line("money", said.proposeBody)]);
        case "graph_apply_proposals": {
            const proposals = records(args, "entities");
            const names = proposals.map((held) => text(held, "name") ?? "").filter((name) => name !== "");
            const lines: Line[] = [line("note", said.applyProposalsBody)];
            if (names.length > 0) {
                lines.unshift(line("target", listed(names)));
            }
            return view(said.applyProposals(proposals.length), lines);
        }
        case "graph_propose_related": {
            const entity = text(args, "entityId");
            const lines: Line[] = [line("money", said.proposeBody)];
            if (entity !== null) {
                lines.unshift(line("target", [`${said.proposeAround} `, ref("entity", entity)]));
            }
            return view(said.proposeRelated, lines);
        }
        default:
            return null;
    }
};
