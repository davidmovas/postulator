export function agentHome(siteId: string | null): string {
    return siteId === null ? "/agent" : `/s/${siteId}/agent`;
}

export function conversationPath(siteId: string | null, conversationId: string): string {
    return `${agentHome(siteId)}/${conversationId}`;
}

export function inboxPath(siteId: string | null): string {
    return `${agentHome(siteId)}/inbox`;
}

const agentRoute = /^(?:\/s\/[^/]+)?\/agent(?:\/|$)/;

export function onAgentScreen(pathname: string): boolean {
    return agentRoute.test(pathname);
}

export function switchedSitePath(pathname: string, siteId: string): string {
    return onAgentScreen(pathname) ? agentHome(siteId) : `/s/${siteId}/overview`;
}
