import { copy } from "../../../../copy/index.js";
import { anchorLabel, decimal } from "../../../templates/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { flag, line, num, record, ref, text, view } from "./card.js";

const said = copy.agent.describe.policies;

function ruleLines(rules: Args | null): Line[] {
    if (rules === null) {
        return [];
    }
    const lines: Line[] = [];
    const maxLinks = num(rules, "maxLinks");
    if (maxLinks !== null) {
        lines.push(line("rule", said.maxLinks(maxLinks)));
    }
    const perTarget = num(rules, "maxPerTarget");
    if (perTarget !== null) {
        lines.push(line("rule", said.perTarget(perTarget)));
    }
    const up = num(rules, "upDepth");
    if (up !== null) {
        lines.push(line("rule", said.upDepth(up)));
    }
    const down = flag(rules, "downLinks");
    if (down !== null) {
        lines.push(line("rule", down ? said.down : said.noDown));
    }
    const siblings = num(rules, "siblingMinWeight");
    if (siblings !== null) {
        lines.push(line("rule", said.siblings(decimal(siblings))));
    }
    const children = flag(rules, "childrenSection");
    if (children !== null) {
        lines.push(line("rule", children ? said.childrenSection : said.noChildrenSection));
    }
    return lines;
}

function policyLines(args: Args): Line[] {
    const lines = ruleLines(record(args, "rules"));
    const external = flag(args, "forbidExternal");
    if (external !== null) {
        lines.push(line("link", external ? said.forbidExternal : said.allowExternal));
    }
    const self = flag(args, "forbidSelf");
    if (self !== null) {
        lines.push(line("link", self ? said.forbidSelf : said.allowSelf));
    }
    const anchors = text(args, "anchorStrategy");
    if (anchors !== null) {
        lines.push(line("rule", said.anchors(anchorLabel(anchors))));
    }
    return lines;
}

export const describePolicies: Describer = (tool, args) => {
    switch (tool) {
        case "policies_create": {
            const lines = policyLines(args);
            const site = text(args, "siteId");
            if (text(args, "scope") === "site" && site !== null) {
                lines.unshift(line("target", [`${copy.agent.describe.templates.scopeSite} `, ref("site", site)]));
            }
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "policies_update": {
            const lines = policyLines(args);
            const name = text(args, "name");
            if (name !== null) {
                lines.unshift(line("field", said.renamed(name)));
            }
            return view([`${said.update} `, ref("policy", text(args, "id") ?? "")], lines);
        }
        case "policies_delete":
            return view([`${said.delete} `, ref("policy", text(args, "id") ?? "")], []);
        default:
            return null;
    }
};
