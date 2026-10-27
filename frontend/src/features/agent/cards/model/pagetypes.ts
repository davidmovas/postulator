import { copy } from "../../../../copy/index.js";
import { anchorLabel, decimal, fieldLabel, presetLabel, providerLabel } from "../../../settings/types/labels.js";
import type { Args, CardView, Describer, Line } from "./card.js";
import { flag, line, listed, num, opOf, record, records, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe.pagetypes;

function ruleLines(rules: Args): Line[] {
    const lines: Line[] = [];
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
    const most = num(rules, "maxLinks");
    if (most !== null) {
        lines.push(line("rule", most === 0 ? said.noCap : said.maxLinks(most)));
    }
    const perTarget = num(rules, "maxPerTarget");
    if (perTarget !== null) {
        lines.push(line("rule", said.perTarget(perTarget)));
    }
    const early = num(rules, "parentLinkWithinParagraphs");
    if (early !== null && early > 0) {
        lines.push(line("rule", said.parentWithin(early)));
    }
    const children = flag(rules, "childrenSection");
    if (children !== null) {
        lines.push(line("rule", children ? said.childrenSection : said.noChildrenSection));
    }
    return lines;
}

function policyLines(policy: Args): Line[] {
    if (flag(policy, "links") === false) {
        return [line("link", said.noLinks)];
    }
    const lines: Line[] = [line("link", said.links)];
    const rules = record(policy, "rules");
    if (rules !== null) {
        lines.push(...ruleLines(rules));
    }
    const external = flag(policy, "forbidExternal");
    if (external !== null) {
        lines.push(line("link", external ? said.forbidExternal : said.allowExternal));
    }
    const self = flag(policy, "forbidSelf");
    if (self !== null) {
        lines.push(line("link", self ? said.forbidSelf : said.allowSelf));
    }
    const anchors = text(policy, "anchorStrategy");
    if (anchors !== null) {
        lines.push(line("rule", said.anchors(anchorLabel(anchors))));
    }
    return lines;
}

function scopeLines(args: Args): Line[] {
    const lines: Line[] = [];
    const enabled = flag(args, "enabled");
    if (enabled !== null) {
        lines.push(line("field", enabled ? said.enabled : said.disabled, enabled ? "ok" : "muted"));
    }
    const ai = flag(args, "aiWrites");
    if (ai !== null) {
        lines.push(line("field", ai ? said.modelWrites : said.personWrites));
    }
    const instruction = text(args, "instruction");
    if (instruction !== null) {
        lines.push(line("note", said.instruction(instruction)));
    }
    const fewest = num(args, "minLength");
    if (fewest !== null) {
        lines.push(line("rule", fewest === 0 ? said.noMinimum : said.minLength(fewest)));
    }
    const most = num(args, "maxLength");
    if (most !== null) {
        lines.push(line("rule", most === 0 ? said.noMaximum : said.maxLength(most)));
    }
    const label = text(args, "label");
    if (label !== null) {
        lines.push(line("field", said.label(label)));
    }
    const provider = text(args, "provider");
    if (provider !== null) {
        lines.push(line("field", said.provider(providerLabel(provider))));
    }
    const links = record(args, "links");
    if (links !== null) {
        lines.push(...policyLines(links));
    }
    return lines;
}

function templateLine(args: Args): Line | null {
    if (!Object.prototype.hasOwnProperty.call(args, "templateId") || args["templateId"] === null) {
        return null;
    }
    const template = text(args, "templateId");
    return template === null ? line("target", said.noTemplate) : line("target", [`${said.template} `, ref("template", template)]);
}

function saveView(args: Args): CardView {
    const lines: Line[] = [];
    const id = text(args, "pageTypeId");
    const name = text(args, "name");
    const template = templateLine(args);
    if (id !== null) {
        if (name !== null) {
            lines.push(line("field", said.renamed(name)));
        }
        if (template !== null) {
            lines.push(template);
        }
        return view([`${said.update} `, ref("pagetype", id)], lines);
    }
    const preset = text(args, "preset");
    if (preset !== null) {
        lines.push(line("field", said.role(presetLabel(preset))));
    }
    if (template !== null) {
        lines.push(template);
    }
    return view(said.create(name ?? ""), lines);
}

function describeTypes(args: Args): CardView | null {
    switch (opOf(args)) {
        case "save":
            return saveView(args);
        case "delete":
            return view(
                [`${said.delete} `, ref("pagetype", text(args, "pageTypeId") ?? "")],
                [line("warn", [`${said.moveTo} `, ref("pagetype", text(args, "reassignTo") ?? "")], "danger")],
            );
        case "scope": {
            const meta = text(args, "metaKey");
            const field = meta === null ? fieldLabel(text(args, "field") ?? "") : (text(args, "label") ?? meta);
            return view([`${said.scope(field)} `, ref("pagetype", text(args, "pageTypeId") ?? "")], scopeLines(args));
        }
        default:
            return null;
    }
}

function range(held: Args | null, phrase: (least: number | null, most: number | null) => string): Line | null {
    if (held === null) {
        return null;
    }
    return line("rule", phrase(num(held, "min"), num(held, "max")));
}

function conditionLines(conditions: Args | null): Line[] {
    const rule = said.conditions;
    if (conditions === null) {
        return [line("rule", rule.none)];
    }
    const lines: Line[] = [];
    const postTypes = strings(conditions, "postTypes");
    if (postTypes.length > 0) {
        lines.push(line("rule", rule.postTypes(listed(postTypes))));
    }
    for (const url of records(conditions, "urls")) {
        lines.push(line("rule", rule.url(text(url, "mode") ?? "", text(url, "value") ?? "")));
    }
    const slugs = strings(conditions, "slugs");
    if (slugs.length > 0) {
        lines.push(line("rule", rule.slugs(listed(slugs))));
    }
    const ancestor = record(conditions, "ancestor");
    if (ancestor !== null) {
        const page = text(ancestor, "pageId");
        const type = text(ancestor, "typeId");
        if (page !== null) {
            lines.push(line("rule", [`${rule.below} `, ref("page", page)]));
        }
        if (type !== null) {
            lines.push(line("rule", [`${rule.belowType} `, ref("pagetype", type)]));
        }
    }
    for (const held of [range(record(conditions, "level"), rule.level), range(record(conditions, "words"), rule.words)]) {
        if (held !== null) {
            lines.push(held);
        }
    }
    const listing = record(conditions, "listing");
    if (listing !== null) {
        lines.push(line("rule", flag(listing, "present") === true ? rule.lists(num(listing, "minProducts") ?? 0) : rule.listsNone));
    }
    const schema = strings(conditions, "schema");
    if (schema.length > 0) {
        lines.push(line("rule", rule.schema(listed(schema))));
    }
    const headings = num(conditions, "minH2");
    if (headings !== null && headings > 0) {
        lines.push(line("rule", rule.minH2(headings)));
    }
    const switches: readonly [string, string, string][] = [
        ["technical", rule.technical, rule.notTechnical],
        ["frontPage", rule.frontPage, rule.notFrontPage],
        ["shop", rule.shop, rule.notShop],
    ];
    for (const [key, yes, no] of switches) {
        const held = flag(conditions, key);
        if (held !== null) {
            lines.push(line("rule", held ? yes : no));
        }
    }
    return lines.length === 0 ? [line("rule", rule.none)] : lines;
}

function describeClassify(args: Args): CardView | null {
    const rules = said.rules;
    switch (opOf(args)) {
        case "apply": {
            const groups = (Array.isArray(args["groups"]) ? args["groups"] : []).filter(
                (held): held is number => typeof held === "number",
            );
            return view(rules.apply, [
                line("list", groups.length === 0 ? rules.allGroups : rules.groups(listed(groups.map(String)))),
                line("note", rules.applyBody),
            ]);
        }
        case "save_rule": {
            const name = text(args, "name") ?? "";
            const lines: Line[] = [line("target", [`${rules.gives} `, ref("pagetype", text(args, "typeId") ?? "")])];
            if (flag(args, "enabled") === false) {
                lines.push(line("warn", rules.off, "warn"));
            }
            lines.push(...conditionLines(record(args, "conditions")));
            return view(text(args, "ruleId") === null ? rules.add(name) : rules.save(name), lines);
        }
        case "delete_rule":
            return view(rules.delete, [line("note", rules.deleteBody)]);
        case "reorder_rules":
            return view(rules.reorder(strings(args, "ruleIds").length), [line("note", rules.reorderBody)]);
        default:
            return null;
    }
}

export const describePageTypes: Describer = (tool, args) => {
    switch (tool) {
        case "pagetypes":
            return describeTypes(args);
        case "pagetypes_classify":
            return describeClassify(args);
        default:
            return null;
    }
};
