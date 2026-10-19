import { copy } from "../../../../copy/index.js";
import { anchorLabel, decimal, fieldLabel, presetLabel, providerLabel } from "../../../settings/types/labels.js";
import type { Args, Describer, Line } from "./card.js";
import { flag, line, num, record, ref, text, view } from "./card.js";

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

export const describePageTypes: Describer = (tool, args) => {
    switch (tool) {
        case "pagetypes_save": {
            const lines: Line[] = [];
            const id = text(args, "id");
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
            const site = text(args, "siteId");
            if (site !== null) {
                lines.push(line("target", [`${said.onSite} `, ref("site", site)]));
            }
            if (template !== null) {
                lines.push(template);
            }
            return view(said.create(name ?? ""), lines);
        }
        case "pagetypes_delete":
            return view(
                [`${said.delete} `, ref("pagetype", text(args, "id") ?? "")],
                [line("warn", [`${said.moveTo} `, ref("pagetype", text(args, "reassignTo") ?? "")], "danger")],
            );
        case "pagetypes_set_scope": {
            const meta = text(args, "metaKey");
            const field = meta === null ? fieldLabel(text(args, "field") ?? "") : (text(args, "label") ?? meta);
            return view([`${said.scope(field)} `, ref("pagetype", text(args, "pageTypeId") ?? "")], scopeLines(args));
        }
        default:
            return null;
    }
};
