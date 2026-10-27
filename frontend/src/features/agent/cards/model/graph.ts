import { copy } from "../../../../copy/index.js";
import type { Keyword } from "../../../../data/types.js";
import { keywordsLine } from "../../../../domain/keywords.js";
import { percent } from "../../../templates/labels.js";
import type { Args, CardView, Describer, Line } from "./card.js";
import { flag, line, listed, num, opOf, records, ref, refs, strings, text, view } from "./card.js";

const said = copy.agent.describe.graph;

const graphEdit = "graph_edit";
const graphPropose = "graph_propose";

const entitiesShown = 6;

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

function factLines(args: Args): Line[] {
    const facts = records(args, "facts");
    return [
        line("list", said.factsCount(facts.length)),
        ...facts.map((fact) => line("field", said.fact(text(fact, "label") ?? "", text(fact, "value") ?? ""))),
    ];
}

function cardLines(args: Args): Line[] {
    const lines: Line[] = [];
    if (typeof args["definition"] === "string") {
        const definition = text(args, "definition");
        lines.push(line("note", definition === null ? said.noDefinition : said.definition(definition)));
    }
    if (Array.isArray(args["facts"])) {
        lines.push(...factLines(args));
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

function entity(args: Args, key = "entityId") {
    return ref("entity", text(args, key) ?? "");
}

function createView(args: Args): CardView {
    const tree = records(args, "entities");
    if (tree.length > 0) {
        const lines: Line[] = [line("note", said.createManyBody)];
        for (const held of tree) {
            const name = text(held, "name") ?? "";
            const parent = text(held, "parentName");
            lines.push(line(parent === null ? "target" : "link", parent === null ? said.root(name) : said.child(name, parent)));
        }
        return view(said.createMany(tree.length), lines);
    }
    const lines = entityFields(args, false);
    const parent = text(args, "parentId");
    lines.unshift(parent === null ? line("target", said.atRoot) : line("target", [`${said.parent} `, ref("entity", parent)]));
    const anchors = records(args, "anchors");
    if (anchors.length > 0) {
        lines.push(line("list", said.anchorsCount(anchors.length)), ...anchorLines(anchors));
    }
    return view(said.create(text(args, "name") ?? ""), lines);
}

function addEdgeView(args: Args): CardView {
    const kind = text(args, "kind") ?? "";
    const lines: Line[] = [];
    const weight = num(args, "weight");
    if (weight !== null) {
        lines.push(line("rule", said.weight(percent(weight))));
    }
    if (text(args, "status") === "proposed") {
        lines.push(line("note", said.proposedEdge));
    }
    const reason = text(args, "reason");
    if (reason !== null) {
        lines.push(line("note", reason));
    }
    return view(
        [`${said.connect} `, entity(args), ` ${kind === "parent" ? said.under : `${said.with} ${kind}`} `, entity(args, "toEntityId")],
        lines,
    );
}

function adoptView(args: Args): CardView {
    const lines: Line[] = [];
    const under = text(args, "under");
    lines.push(under === null ? line("target", said.adoptWhole) : line("target", [`${said.adoptUnder} `, ref("page", under)]));
    const chosen = strings(args, "entityIds");
    if (chosen.length > 0) {
        lines.push(line("list", [`${said.adoptOnly} `, ...refs("entity", chosen, entitiesShown)]));
    }
    lines.push(line("note", said.adoptBody));
    return view(said.adopt, lines);
}

function applyView(args: Args): CardView {
    const proposals = records(args, "proposals");
    const names = proposals.map((held) => text(held, "name") ?? "").filter((name) => name !== "");
    const lines: Line[] = [line("note", said.applyProposalsBody)];
    if (names.length > 0) {
        lines.unshift(line("target", listed(names)));
    }
    if (text(args, "proposalsOf") === "site") {
        lines.push(line("note", said.fromSite));
    }
    return view(said.applyProposals(proposals.length), lines);
}

function describeEdit(args: Args): CardView | null {
    switch (opOf(args)) {
        case "create":
            return createView(args);
        case "update":
            return view([`${said.update} `, entity(args)], entityFields(args, true));
        case "delete":
            return view([`${said.delete} `, entity(args)], [line("warn", said.deleteBody, "danger")]);
        case "anchors": {
            const anchors = records(args, "anchors");
            return view([`${said.setAnchors} `, entity(args)], [line("list", said.anchorsCount(anchors.length)), ...anchorLines(anchors)]);
        }
        case "add_edge":
            return addEdgeView(args);
        case "approve_edge":
            return view([`${said.approve} `, ref("edge", text(args, "edgeId") ?? "")], []);
        case "reject_edge":
            return view([`${said.reject} `, ref("edge", text(args, "edgeId") ?? "")], []);
        case "delete_edge":
            return view([`${said.deleteEdge} `, ref("edge", text(args, "edgeId") ?? "")], []);
        case "move":
            return view(
                [`${said.move} `, entity(args), ` ${said.under} `, entity(args, "parentId")],
                flag(args, "keepBoth") === true ? [line("note", said.keepBoth)] : [],
            );
        case "rescore":
            return view(said.recompute, [line("note", said.recomputeBody)]);
        case "adopt_parents":
            return adoptView(args);
        case "apply_proposals":
            return applyView(args);
        default:
            return null;
    }
}

function cardsView(args: Args): CardView {
    const cards = records(args, "cards");
    const lines: Line[] = [line("note", said.proposeCardsBody)];
    for (const card of cards) {
        lines.push(line("target", [entity(card)]));
        const definition = text(card, "definition");
        if (definition !== null) {
            lines.push(line("note", said.definition(definition)));
        }
        if (Array.isArray(card["facts"])) {
            lines.push(...factLines(card));
        }
        const reason = text(card, "reason");
        if (reason !== null) {
            lines.push(line("note", said.proposeCardsReason(reason)));
        }
    }
    return view(said.proposeCards(cards.length), lines);
}

function scopeLines(args: Args): Line[] {
    const lines: Line[] = [];
    const pages = strings(args, "pageIds");
    if (pages.length > 0) {
        lines.push(line("target", [`${said.onlyPages} `, ...refs("page", pages, entitiesShown)]));
    }
    const prefix = text(args, "pathPrefix");
    if (prefix !== null) {
        lines.push(line("target", said.pathPrefix(prefix)));
    }
    return lines;
}

function describePropose(args: Args): CardView | null {
    switch (opOf(args)) {
        case "pages":
            return view(said.proposeFromPages, [...scopeLines(args), line("money", said.proposeBody)]);
        case "preview":
            return view(said.previewFromPages, [
                ...scopeLines(args),
                line(text(args, "source") === "site" ? "note" : "money", text(args, "source") === "site" ? said.fromSite : said.proposeBody),
            ]);
        case "keywords": {
            const phrases = strings(args, "keywords");
            const lines: Line[] = [line("list", listed(phrases))];
            const parent = text(args, "parentEntityId");
            if (parent !== null) {
                lines.push(line("target", [`${said.parent} `, ref("entity", parent)]));
            }
            lines.push(line("money", said.proposeKeywordsBody));
            return view(said.proposeFromKeywords(phrases.length), lines);
        }
        case "related": {
            const around = text(args, "entityId");
            const lines: Line[] = [line("money", said.proposeBody)];
            if (around !== null) {
                lines.unshift(line("target", [`${said.proposeAround} `, ref("entity", around)]));
            }
            return view(said.proposeRelated, lines);
        }
        case "cards":
            return cardsView(args);
        default:
            return null;
    }
}

export const describeGraph: Describer = (tool, args) => {
    switch (tool) {
        case graphEdit:
            return describeEdit(args);
        case graphPropose:
            return describePropose(args);
        default:
            return null;
    }
};
