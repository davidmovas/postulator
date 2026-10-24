import { copy } from "../../../../copy/index.js";
import type { Keyword } from "../../../../data/types.js";
import { keywordsLine } from "../../../../domain/keywords.js";
import { percent } from "../../../templates/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { line, listed, num, records, ref, text, view } from "./card.js";

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
    const intent = text(args, "intent");
    if (intent !== null) {
        lines.push(line("field", said.intent(intent)));
    }
    const keywords = keywordsOf(args);
    if (keywords.length > 0) {
        lines.push(line("target", said.keywords(keywordsLine(keywords))));
    }
    lines.push(...cardLines(args));
    return lines;
}

function cardLines(args: Args): Line[] {
    const lines: Line[] = [];
    if (typeof args["definition"] === "string") {
        const definition = text(args, "definition");
        lines.push(line("note", definition === null ? said.noDefinition : said.definition(definition)));
    }
    if (Array.isArray(args["facts"])) {
        const facts = records(args, "facts");
        lines.push(line("list", said.factsCount(facts.length)));
        lines.push(...facts.map((fact) => line("field", said.fact(text(fact, "label") ?? "", text(fact, "value") ?? ""))));
    }
    if (Array.isArray(args["notes"])) {
        const notes = records(args, "notes");
        lines.push(line("list", said.notesCount(notes.length)));
        lines.push(...notes.map((note) => line("note", said.note(text(note, "label") ?? ""))));
    }
    return lines;
}

function keywordsOf(args: Args): Keyword[] {
    return records(args, "keywords").map((held) => {
        const volume = num(held, "volume");
        return volume === null ? { text: text(held, "text") ?? "" } : { text: text(held, "text") ?? "", volume };
    });
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
        case "graph_propose_cards": {
            const cards = records(args, "cards");
            const lines: Line[] = [line("note", said.proposeCardsBody)];
            for (const card of cards) {
                lines.push(line("target", [ref("entity", text(card, "entityId") ?? "")]));
                const definition = text(card, "definition");
                if (definition !== null) {
                    lines.push(line("note", said.definition(definition)));
                }
                if (Array.isArray(card["facts"])) {
                    const facts = records(card, "facts");
                    lines.push(line("list", said.factsCount(facts.length)));
                    lines.push(...facts.map((fact) => line("field", said.fact(text(fact, "label") ?? "", text(fact, "value") ?? ""))));
                }
                const reason = text(card, "reason");
                if (reason !== null) {
                    lines.push(line("note", said.proposeCardsReason(reason)));
                }
            }
            return view(said.proposeCards(cards.length), lines);
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
