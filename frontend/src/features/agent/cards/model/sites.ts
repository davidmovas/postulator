import { copy } from "../../../../copy/index.js";
import type { Describer, Line } from "./card.js";
import { flag, line, masked, ref, text, view } from "./card.js";

const said = copy.agent.describe.sites;

export const describeSites: Describer = (tool, args) => {
    switch (tool) {
        case "sites_create": {
            const lines: Line[] = [];
            const url = text(args, "baseUrl");
            if (url !== null) {
                lines.push(line("link", said.address(url)));
            }
            const user = text(args, "username");
            if (user !== null) {
                lines.push(line("key", said.user(user)));
            }
            if (text(args, "password") !== null) {
                lines.push(line("key", said.password));
            }
            if (flag(args, "allowInsecure") === true) {
                lines.push(line("warn", said.insecure, "warn"));
            }
            return view(said.create(text(args, "name") ?? ""), lines);
        }
        case "sites_update": {
            const lines: Line[] = [];
            const name = text(args, "name");
            if (name !== null) {
                lines.push(line("field", said.renamed(name)));
            }
            const url = text(args, "baseUrl");
            if (url !== null) {
                lines.push(line("link", said.address(url)));
            }
            const user = text(args, "username");
            if (user !== null) {
                lines.push(line("key", said.user(user)));
            }
            const password = text(args, "password");
            if (password !== null || masked(String(args["password"] ?? ""))) {
                lines.push(line("key", said.password));
            }
            const insecure = flag(args, "allowInsecure");
            if (insecure !== null) {
                lines.push(line("warn", insecure ? said.insecure : said.secure, insecure ? "warn" : "muted"));
            }
            const status = text(args, "status");
            if (status !== null) {
                lines.push(line("field", said.status(status)));
            }
            const template = text(args, "defaultTemplateId");
            if (template !== null) {
                lines.push(line("target", [`${said.defaultTemplate} `, ref("template", template)]));
            }
            const policy = text(args, "defaultLinkPolicyId");
            if (policy !== null) {
                lines.push(line("target", [`${said.defaultPolicy} `, ref("policy", policy)]));
            }
            return view([`${said.update} `, ref("site", text(args, "id") ?? "")], lines);
        }
        case "sites_delete":
            return view([`${said.delete} `, ref("site", text(args, "id") ?? "")], [line("warn", said.deleteBody, "danger")]);
        default:
            return null;
    }
};
