import type { SiteMapNode, SiteMapRef } from "../../../data/types.js";

export function aRef(pageId: string, path: string, overrides: Partial<SiteMapRef> = {}): SiteMapRef {
    return {
        pageId,
        path,
        title: path,
        type: "Hub",
        preset: "hub" as SiteMapRef["preset"],
        role: "hub" as SiteMapRef["role"],
        onSite: true,
        ...overrides,
    };
}

export function aNode(pageId: string, path: string, overrides: Partial<SiteMapNode> = {}): SiteMapNode {
    return {
        page: aRef(pageId, path),
        status: "published" as SiteMapNode["status"],
        parentReason: "jsonld_breadcrumb" as SiteMapNode["parentReason"],
        parentMethod: "href" as SiteMapNode["parentMethod"],
        technical: "" as SiteMapNode["technical"],
        readOnly: "" as SiteMapNode["readOnly"],
        body: "client" as SiteMapNode["body"],
        linksIn: 0,
        linksOut: 0,
        findings: 0,
        children: 0,
        descendants: 0,
        ...overrides,
    };
}
