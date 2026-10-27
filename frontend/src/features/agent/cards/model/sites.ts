import { copy } from "../../../../copy/index.js";
import type { Args, Describer, Line, Part } from "./card.js";
import { flag, has, line, listed, masked, num, opOf, records, ref, strings, text, view } from "./card.js";

const said = copy.agent.describe.sites;

const sitesManage = "sites_manage";

function sitePart(args: Args): Part {
    const id = text(args, "siteId");
    return id === null ? said.thisSite : ref("site", id);
}

function listLine(args: Args, key: string, phrase: (items: string) => string): Line | null {
    if (!has(args, key)) {
        return null;
    }
    const items = strings(args, key);
    return line("rule", phrase(items.length === 0 ? said.defaults : listed(items)));
}

function settingsLines(args: Args): Line[] {
    const lines: Line[] = [];
    const agent = text(args, "userAgent");
    if (agent !== null) {
        lines.push(line("field", said.userAgent(agent)));
    }
    const renders = num(args, "renderConcurrency");
    if (renders !== null) {
        lines.push(line("rule", said.renders(renders)));
    }
    const walk = num(args, "fullWalkEvery");
    if (walk !== null) {
        lines.push(line("rule", said.fullWalk(walk)));
    }
    const form = text(args, "linkUrlForm");
    if (form !== null) {
        lines.push(line("link", said.linkForm(form)));
    }
    for (const held of [
        listLine(args, "technicalShortcodes", said.shortcodes),
        listLine(args, "technicalBlocks", said.blocks),
    ]) {
        if (held !== null) {
            lines.push(held);
        }
    }
    if (has(args, "builders")) {
        const names = records(args, "builders").map((builder) => text(builder, "name") ?? "");
        lines.push(line("rule", said.builders(names.length === 0 ? said.defaults : listed(names))));
    }
    return lines;
}

function credentialLines(args: Args): Line[] {
    const lines: Line[] = [];
    const url = text(args, "baseUrl");
    if (url !== null) {
        lines.push(line("link", said.address(url)));
    }
    const user = text(args, "username");
    if (user !== null) {
        lines.push(line("key", said.user(user)));
    }
    if (text(args, "password") !== null || masked(String(args["password"] ?? ""))) {
        lines.push(line("key", said.password));
    }
    const insecure = flag(args, "allowInsecure");
    if (insecure !== null) {
        lines.push(line("warn", insecure ? said.insecure : said.secure, insecure ? "warn" : "muted"));
    }
    return lines;
}

export const describeSites: Describer = (tool, args) => {
    if (tool !== sitesManage) {
        return null;
    }
    switch (opOf(args)) {
        case "create":
            return view(said.create(text(args, "name") ?? ""), credentialLines(args));
        case "update": {
            const lines: Line[] = [];
            const name = text(args, "name");
            if (name !== null) {
                lines.push(line("field", said.renamed(name)));
            }
            lines.push(...credentialLines(args));
            const status = text(args, "status");
            if (status !== null) {
                lines.push(line("field", said.status(status)));
            }
            const template = text(args, "defaultTemplateId");
            if (template !== null) {
                lines.push(line("target", [`${said.defaultTemplate} `, ref("template", template)]));
            }
            lines.push(...settingsLines(args));
            return view([`${said.update} `, sitePart(args)], lines);
        }
        case "delete":
            return view([`${said.delete} `, sitePart(args)], [line("warn", said.deleteBody, "danger")]);
        case "sync": {
            const lines: Line[] = [];
            if (flag(args, "full") === true) {
                lines.push(line("rule", said.syncFull));
            }
            if (flag(args, "renderRescan") === true) {
                lines.push(line("rule", said.syncRescan));
            }
            lines.push(line("note", said.syncBody));
            return view([`${said.sync} `, sitePart(args)], lines);
        }
        case "check_plugin":
            return view([`${said.checkPlugin} `, sitePart(args)], [line("note", said.checkPluginBody)]);
        default:
            return null;
    }
};
