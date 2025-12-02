import { SitemapNode, CreateNodeInput, NodeSource } from "@/models/sitemaps";

interface PathSegment {
    slug: string;
    title: string;
    fullPath: string;
    depth: number;
}

interface ParsedPathLine {
    path: string;
    customTitle?: string;
}

/**
 * Parse a line that may contain a path and optional title
 * Format: "/path/to/page Optional Title Here"
 * The title is everything after the first space that follows the path
 */
export function parsePathLine(line: string): ParsedPathLine {
    const trimmed = line.trim();
    if (!trimmed) return { path: "" };

    // Ensure path starts with /
    const normalized = trimmed.startsWith("/") ? trimmed : "/" + trimmed;

    // Find where the path ends (first space after path segments)
    // Path segments only contain: a-z, 0-9, -, /
    const match = normalized.match(/^(\/[a-zA-Z0-9\-\/]+)(?:\s+(.+))?$/);

    if (match) {
        return {
            path: match[1].toLowerCase(),
            customTitle: match[2]?.trim() || undefined,
        };
    }

    // Fallback: treat the whole thing as a path
    return { path: normalized.split(/\s+/)[0].toLowerCase() };
}

/**
 * Parse a URL path into segments
 * e.g., "/page-1/page-2/page-3" -> [{slug: "page-1", ...}, {slug: "page-2", ...}, ...]
 */
export function parsePathSegments(path: string, customTitle?: string): PathSegment[] {
    // Normalize path - remove leading/trailing slashes and whitespace
    const normalized = path.trim().replace(/^\/+|\/+$/g, "");
    if (!normalized) return [];

    const segments = normalized.split("/").filter((s) => s.length > 0);
    const result: PathSegment[] = [];

    let currentPath = "";
    segments.forEach((slug, index) => {
        // Normalize slug
        const normalizedSlug = slug
            .toLowerCase()
            .replace(/[^a-z0-9-]/g, "-")
            .replace(/-+/g, "-")
            .replace(/^-|-$/g, "");

        if (!normalizedSlug) return;

        currentPath += "/" + normalizedSlug;

        const isLastSegment = index === segments.length - 1;

        result.push({
            slug: normalizedSlug,
            // Use custom title only for the last segment
            title: isLastSegment && customTitle
                ? customTitle
                : generateTitleFromSlug(normalizedSlug),
            fullPath: currentPath,
            depth: index,
        });
    });

    return result;
}

/**
 * Generate a placeholder title from a slug
 * e.g., "my-page-title" -> "My Page Title"
 */
function generateTitleFromSlug(slug: string): string {
    return slug
        .split("-")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" ");
}

interface NodeCreationPlan {
    // Nodes that need to be created (in order, parent first)
    toCreate: Array<{
        slug: string;
        title: string;
        parentSlug: string | null; // null means root or orphan
        parentFullPath: string | null;
    }>;
    // Mapping from full path to existing node
    existingNodes: Map<string, SitemapNode>;
}

/**
 * Build a plan for creating nodes from multiple paths
 * This handles:
 * - Deduplication of paths
 * - Finding existing nodes to connect to
 * - Creating nodes in correct order (parents before children)
 * - Optional custom titles for leaf nodes
 */
export function buildNodeCreationPlan(
    pathLines: string[],
    existingNodes: SitemapNode[]
): NodeCreationPlan {
    // Build a map of existing nodes by their full path
    const existingByPath = new Map<string, SitemapNode>();
    const nodePathMap = buildNodePathMap(existingNodes);

    nodePathMap.forEach((path, nodeId) => {
        const node = existingNodes.find((n) => n.id === nodeId);
        if (node) {
            existingByPath.set(path, node);
        }
    });

    // Track all segments we need to create
    // Key is fullPath, value includes title info
    const segmentsToCreate = new Map<string, {
        slug: string;
        title: string;
        parentFullPath: string | null;
    }>();

    // Process each path line
    for (const line of pathLines) {
        const { path, customTitle } = parsePathLine(line);
        if (!path) continue;

        const segments = parsePathSegments(path, customTitle);

        for (let i = 0; i < segments.length; i++) {
            const segment = segments[i];
            const parentFullPath = i > 0 ? segments[i - 1].fullPath : null;

            // Skip if this exact path already exists
            if (existingByPath.has(segment.fullPath)) {
                continue;
            }

            // If we've already planned to create this, only update title if this is a leaf with custom title
            if (segmentsToCreate.has(segment.fullPath)) {
                const existing = segmentsToCreate.get(segment.fullPath)!;
                const isLeaf = i === segments.length - 1;
                // Update title if this line has a custom title for the leaf
                if (isLeaf && customTitle) {
                    existing.title = customTitle;
                }
                continue;
            }

            segmentsToCreate.set(segment.fullPath, {
                slug: segment.slug,
                title: segment.title,
                parentFullPath,
            });
        }
    }

    // Convert to array and sort by depth (parents first)
    const toCreate = Array.from(segmentsToCreate.entries())
        .sort((a, b) => {
            const depthA = a[0].split("/").length;
            const depthB = b[0].split("/").length;
            return depthA - depthB;
        })
        .map(([, data]) => ({
            ...data,
            parentSlug: data.parentFullPath
                ? data.parentFullPath.split("/").pop() || null
                : null,
        }));

    return {
        toCreate,
        existingNodes: existingByPath,
    };
}

/**
 * Build a map from node ID to its full path
 */
function buildNodePathMap(nodes: SitemapNode[]): Map<number, string> {
    const pathMap = new Map<number, string>();
    const nodeMap = new Map<number, SitemapNode>();

    nodes.forEach((node) => nodeMap.set(node.id, node));

    function getNodePath(node: SitemapNode): string {
        if (node.isRoot) {
            return "";
        }

        if (pathMap.has(node.id)) {
            return pathMap.get(node.id)!;
        }

        let path: string;
        if (node.parentId) {
            const parent = nodeMap.get(node.parentId);
            if (parent) {
                const parentPath = getNodePath(parent);
                path = parentPath + "/" + node.slug;
            } else {
                path = "/" + node.slug;
            }
        } else {
            path = "/" + node.slug;
        }

        pathMap.set(node.id, path);
        return path;
    }

    nodes.forEach((node) => getNodePath(node));
    return pathMap;
}

/**
 * Find a node by its full path
 */
export function findNodeByPath(
    path: string,
    nodes: SitemapNode[]
): SitemapNode | undefined {
    const pathMap = buildNodePathMap(nodes);

    for (const [nodeId, nodePath] of pathMap) {
        if (nodePath === path) {
            return nodes.find((n) => n.id === nodeId);
        }
    }

    return undefined;
}

/**
 * Create nodes from a list of path lines
 * Each line can be: "/path/to/page" or "/path/to/page Custom Title"
 * Returns an async generator that yields progress updates
 */
export async function* createNodesFromPaths(
    pathLines: string[],
    sitemapId: number,
    existingNodes: SitemapNode[],
    createNode: (input: CreateNodeInput) => Promise<SitemapNode>
): AsyncGenerator<{ created: number; total: number; currentPath: string }> {
    const plan = buildNodeCreationPlan(pathLines, existingNodes);

    if (plan.toCreate.length === 0) {
        return;
    }

    // Track newly created nodes by their full path
    const createdNodes = new Map<string, SitemapNode>(plan.existingNodes);
    let created = 0;
    const total = plan.toCreate.length;

    // Find root node
    const rootNode = existingNodes.find((n) => n.isRoot);

    for (const item of plan.toCreate) {
        let parentId: number | undefined;

        if (item.parentFullPath) {
            // Find parent in existing or newly created nodes
            const parent = createdNodes.get(item.parentFullPath);
            if (parent) {
                parentId = parent.id;
            }
        } else if (rootNode) {
            // First-level nodes connect to root
            parentId = rootNode.id;
        }

        // Get max position for siblings
        const siblings = existingNodes.filter((n) => n.parentId === parentId);
        const maxPosition = siblings.reduce((max, n) => Math.max(max, n.position), 0);

        const input: CreateNodeInput = {
            sitemapId,
            parentId,
            title: item.title,
            slug: item.slug,
            position: maxPosition + 1,
            source: "manual" as NodeSource,
        };

        const newNode = await createNode(input);

        // Add to created nodes map
        const fullPath = item.parentFullPath
            ? item.parentFullPath + "/" + item.slug
            : "/" + item.slug;
        createdNodes.set(fullPath, newNode);

        // Also add to existingNodes for position calculation
        existingNodes.push(newNode);

        created++;
        yield { created, total, currentPath: fullPath };
    }
}
