import type { KeyboardEvent, ReactElement } from "react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";

import { CanvasHost } from "../../../canvas/host.js";
import type { CanvasHandle, PointerInfo, WheelInfo } from "../../../canvas/host.js";
import { HitGrid } from "../../../canvas/hit-grid.js";
import { readPalette } from "../../../canvas/palette.js";
import type { Palette } from "../../../canvas/palette.js";
import { TextCache } from "../../../canvas/text-cache.js";
import { layoutTree } from "../../../canvas/tree-layout.js";
import type { Layout, Placed } from "../../../canvas/tree-layout.js";
import { centerOn, fit, panBy, toWorld, visibleWorld, zoomAt } from "../../../canvas/viewport.js";
import type { Point, Size, Viewport } from "../../../canvas/viewport.js";
import { copy } from "../../../copy/index.js";
import { CallSplitIcon, LinkOffIcon } from "../../../ui/index.js";
import type { Severity } from "../../links/model/audit.js";
import { entityIcon, kindTone } from "../labels.js";
import type { FoldState, VisibleRow } from "../model/fold.js";
import { childLimit, treeOf } from "../model/fold.js";
import { nodeStateOf } from "../model/index.js";
import type { GraphIndex, NodeState } from "../model/index.js";
import { detailOf, labelled } from "../model/lod.js";
import { move } from "../model/navigation.js";
import type { NavKey } from "../model/navigation.js";
import { setViewport, viewportOf } from "../state.js";
import { Minimap } from "./minimap.js";
import { arc, chevron, connector, drawIcon, roundRect, toneColor } from "./render.js";

export const rowHeight = 28;
export const nodeHeight = 24;
const columnGap = 48;
const rootGap = 1;
const padding = 8;
const iconSize = 14;
const badgeSize = 12;
const badgeGap = 6;
const labelMax = 220;
const fitPadding = 40;
const dragThreshold = 3;
const hitCell = 64;

const layoutOptions = { rowHeight, nodeHeight, columnGap, rootGap };

interface Fonts {
    label: string;
    chip: string;
    faint: string;
}

interface Measure {
    text: string;
    chip: string;
    chipWidth: number;
    badges: number;
    width: number;
}

interface Drag {
    start: Point;
    view: Viewport;
    moved: boolean;
    target: Placed | null;
    mode: "pan" | "node";
    cursor: Point;
    over: string | null;
}

export interface GraphMapProps {
    siteId: string;
    index: GraphIndex;
    rows: readonly VisibleRow[];
    fold: FoldState;
    selectedId: string | null;
    matched: ReadonlySet<string> | null;
    showRelated: boolean;
    highlightEdgeId: string | null;
    pulse: Pulse | null;
    tint: ReadonlyMap<string, Severity> | null;
    revealVersion: number;
    onSelect: (id: string | null) => void;
    onPick: (id: string) => void;
    onToggle: (id: string) => void;
    onLiftMore: (parentId: string, count: number) => void;
    onContextMenu?: (id: string | null, screen: Point) => void;
    onHover?: (id: string | null, screen: Point) => void;
    onCreateChild: (id: string | null) => void;
    onConnect: (id: string) => void;
    onDelete: (id: string) => void;
    onReparent: (childId: string, parentId: string) => void;
    minimap: boolean;
    ref?: (handle: MapHandle | null) => void;
}

export interface MapHandle {
    fit(): void;
    zoomBy(factor: number): void;
}

export interface Pulse {
    ids: ReadonlySet<string>;
    until: number;
}

export const pulseMs = 3000;

function fontsOf(palette: Palette): Fonts {
    return {
        label: `500 12px ${palette.fontSans}`,
        chip: `500 10px ${palette.fontMono}`,
        faint: `400 11px ${palette.fontSans}`,
    };
}

export function GraphMap({
    siteId,
    index,
    rows,
    fold,
    selectedId,
    matched,
    showRelated,
    highlightEdgeId,
    pulse,
    tint,
    revealVersion,
    onSelect,
    onPick,
    onToggle,
    onLiftMore,
    onContextMenu,
    onHover,
    onCreateChild,
    onConnect,
    onDelete,
    onReparent,
    minimap,
    ref,
}: GraphMapProps): ReactElement {
    const host = useRef<CanvasHandle>(null);
    const frame = useRef<HTMLDivElement>(null);
    const [palette, setPalette] = useState<Palette | null>(null);
    const fonts = useMemo(() => (palette === null ? null : fontsOf(palette)), [palette]);
    const measurer = useMemo(() => document.createElement("canvas").getContext("2d"), []);
    const texts = useMemo(() => (measurer === null ? null : new TextCache(measurer)), [measurer]);
    const hover = useRef<string | null>(null);
    const drag = useRef<Drag | null>(null);
    const view = useRef<Viewport | null>(viewportOf(siteId));
    const size = useRef<Size>({ width: 0, height: 0 });

    useLayoutEffect(() => {
        if (frame.current !== null) {
            setPalette(readPalette(frame.current));
        }
    }, []);

    const [measureVersion, setMeasureVersion] = useState(0);

    useEffect(() => {
        if (texts === null || typeof document.fonts === "undefined") {
            return undefined;
        }
        let live = true;
        void document.fonts.ready.then(() => {
            if (!live) {
                return;
            }
            texts.clear();
            setMeasureVersion((held) => held + 1);
        });
        return () => {
            live = false;
        };
    }, [texts]);

    const measures = useMemo(() => {
        const out = new Map<string, Measure>();
        if (texts === null || fonts === null) {
            return out;
        }
        for (const row of rows) {
            if (row.kind === "more") {
                const text = `${copy.graph.node.showMore(Math.min(childLimit, row.hidden))} · ${copy.graph.node.more(row.shown, row.hidden)}`;
                out.set(row.id, { text, chip: "", chipWidth: 0, badges: 0, width: padding * 2 + texts.width(fonts.faint, text) });
                continue;
            }
            const held = index.byId.get(row.id);
            const flags = index.problems.get(row.id);
            const text = texts.ellipsise(fonts.label, held?.name ?? row.id, labelMax);
            const badges =
                (flags?.noPage ? badgeSize + 4 : 0) + (flags?.multiParent ? badgeSize + 4 : 0) + ((flags?.proposed ?? 0) > 0 ? 22 : 0);
            const chip = row.childCount > 0 ? String(row.expanded ? "" : row.hiddenChildren) : "";
            const chipWidth = row.childCount > 0 ? 14 + (chip === "" ? 0 : texts.width(fonts.chip, chip) + 4) : 0;
            const width =
                padding +
                iconSize +
                6 +
                texts.width(fonts.label, text) +
                (badges > 0 ? badgeGap + badges : 0) +
                (chipWidth > 0 ? badgeGap + chipWidth : 0) +
                padding;
            out.set(row.id, { text, chip, chipWidth, badges, width });
        }
        return out;
    }, [rows, index, texts, fonts, measureVersion]);

    const layout: Layout = useMemo(
        () => layoutTree(treeOf(rows, (row) => measures.get(row.id)?.width ?? 40), layoutOptions),
        [rows, measures],
    );
    const grid = useMemo(() => new HitGrid(layout.nodes, hitCell), [layout]);
    const rowById = useMemo(() => new Map(rows.map((row) => [row.id, row])), [rows]);

    const latest = useRef({ index, rows, fold, selectedId, matched, showRelated, highlightEdgeId, pulse, tint, layout, grid, rowById, measures, palette, fonts });
    latest.current = { index, rows, fold, selectedId, matched, showRelated, highlightEdgeId, pulse, tint, layout, grid, rowById, measures, palette, fonts };

    useEffect(() => {
        if (pulse === null) {
            return undefined;
        }
        const timer = window.setInterval(() => {
            host.current?.redraw();
            if (Date.now() >= pulse.until) {
                window.clearInterval(timer);
            }
        }, 80);
        return () => {
            window.clearInterval(timer);
        };
    }, [pulse]);

    const commitView = useCallback(
        (next: Viewport): void => {
            view.current = next;
            setViewport(siteId, next);
            host.current?.redraw();
        },
        [siteId],
    );

    const fitAll = useCallback((): void => {
        commitView(fit(latest.current.layout.bounds, size.current, fitPadding));
    }, [commitView]);

    useEffect(() => {
        ref?.({
            fit: fitAll,
            zoomBy: (factor) => {
                const current = view.current;
                if (current !== null) {
                    commitView(zoomAt(current, { x: size.current.width / 2, y: size.current.height / 2 }, factor));
                }
            },
        });
        return () => {
            ref?.(null);
        };
    }, [ref, fitAll, commitView]);

    useEffect(() => {
        if (view.current === null) {
            if (palette !== null && layout.nodes.length > 0 && size.current.width > 0) {
                fitAll();
            }
            return;
        }
        host.current?.redraw();
    }, [layout, fitAll, selectedId, matched, showRelated, highlightEdgeId, tint, palette]);

    useEffect(() => {
        if (selectedId === null || view.current === null) {
            return;
        }
        const placed = layout.byId.get(selectedId);
        if (placed === undefined) {
            return;
        }
        const seen = visibleWorld(view.current, size.current);
        const inside =
            placed.x >= seen.x && placed.x + placed.width <= seen.x + seen.width && placed.y >= seen.y && placed.y + placed.height <= seen.y + seen.height;
        if (!inside) {
            commitView(centerOn(view.current, { x: placed.x + placed.width / 2, y: placed.y + placed.height / 2 }, size.current));
        }
    }, [revealVersion, selectedId, layout, commitView]);

    const draw = useCallback((context: CanvasRenderingContext2D, area: Size): void => {
        const { layout: placed, grid: hits, rowById: byRow, measures: sizes, palette: colors, fonts: faces, index: graph } = latest.current;
        const { selectedId: selected, matched: lit, showRelated: allRelated, highlightEdgeId: highlighted, pulse: pulsing, tint: proof } = latest.current;
        const now = Date.now();
        const glow = pulsing !== null && now < pulsing.until ? (pulsing.until - now) / pulseMs : 0;
        if (colors === null || faces === null) {
            return;
        }
        const current = view.current;
        if (current === null) {
            return;
        }
        const proofFill: Readonly<Record<Severity, string>> = { ok: colors.okSoft, warn: colors.warnSoft, danger: colors.dangerSoft, muted: colors.mutedSoft };
        const proofStroke: Readonly<Record<Severity, string>> = { ok: colors.ok, warn: colors.warn, danger: colors.danger, muted: colors.hairline };
        const stateFill: Readonly<Record<NodeState, string>> = {
            mismatch: colors.dangerSoft,
            working: colors.accentSoft,
            published: colors.okSoft,
            exists: colors.warnSoft,
            planned: colors.infoSoft,
            archived: colors.mutedSoft,
            noPage: colors.inset,
        };
        const stateStroke: Readonly<Record<NodeState, string>> = {
            mismatch: colors.danger,
            working: colors.accent,
            published: colors.ok,
            exists: colors.warn,
            planned: colors.info,
            archived: colors.muted,
            noPage: colors.hairline,
        };
        const detail = detailOf(current.k);
        const seen = visibleWorld(current, area);
        const reach = { x: seen.x - 640, y: seen.y - rowHeight * 2, width: seen.width + 1280, height: seen.height + rowHeight * 4 };
        const nodes = hits.within(reach);

        context.save();
        context.translate(current.x, current.y);
        context.scale(current.k, current.k);

        if (current.k >= 0.7) {
            context.fillStyle = colors.hairline;
            const step = rowHeight;
            const startX = Math.floor(seen.x / step) * step;
            const startY = Math.floor(seen.y / step) * step;
            for (let x = startX; x < seen.x + seen.width; x += step) {
                for (let y = startY; y < seen.y + seen.height; y += step) {
                    context.fillRect(x, y, 1, 1);
                }
            }
        }

        const hovered = hover.current;
        const neighbours = new Set<string>();
        for (const focus of [selected, hovered]) {
            if (focus === null) {
                continue;
            }
            for (const link of graph.related.get(focus) ?? []) {
                neighbours.add(link.otherId);
            }
        }

        for (const node of nodes) {
            if (node.parentId === null) {
                continue;
            }
            const parent = placed.byId.get(node.parentId);
            if (parent === undefined) {
                continue;
            }
            const row = byRow.get(node.id);
            const proposedPlacement = row?.kind === "entity" && graph.placementProposed.has(node.id);
            const hot = node.id === selected || node.id === hovered || parent.id === selected || parent.id === hovered;
            const dim = lit !== null && !lit.has(node.id) && !lit.has(parent.id);
            context.globalAlpha = dim ? 0.25 : 1;
            connector(
                context,
                { x: parent.x + parent.width, y: parent.y + nodeHeight / 2 },
                { x: node.x, y: node.y + nodeHeight / 2 },
                proposedPlacement ? colors.info : hot ? colors.accent : row?.kind === "more" ? colors.hairline : colors.edge,
                hot ? 1.5 : 1,
                proposedPlacement ? [4, 3] : [],
            );
            context.globalAlpha = 1;
        }

        const drawn = new Set<string>();
        const arcsFor = (id: string): void => {
            const from = placed.byId.get(id);
            if (from === undefined) {
                return;
            }
            for (const link of graph.related.get(id) ?? []) {
                if (drawn.has(link.edgeId)) {
                    continue;
                }
                const to = placed.byId.get(link.otherId);
                if (to === undefined) {
                    continue;
                }
                drawn.add(link.edgeId);
                const strong = id === selected || id === hovered || link.otherId === selected || link.otherId === hovered;
                context.globalAlpha = strong ? 0.9 : 0.45;
                arc(
                    context,
                    { x: from.x + from.width / 2, y: from.y + nodeHeight },
                    { x: to.x + to.width / 2, y: to.y + nodeHeight },
                    colors.info,
                    strong ? 1.5 : 1,
                    link.status === "proposed" ? [4, 3] : [],
                );
                context.globalAlpha = 1;
            }
        };
        if (allRelated) {
            for (const node of nodes) {
                arcsFor(node.id);
            }
        } else {
            for (const focus of [selected, hovered]) {
                if (focus !== null) {
                    arcsFor(focus);
                }
            }
        }

        const litEnds = new Set<string>();
        const litEdge = highlighted === null ? undefined : graph.edgeById.get(highlighted);
        if (litEdge !== undefined) {
            litEnds.add(litEdge.fromEntityId);
            litEnds.add(litEdge.toEntityId);
            const from = placed.byId.get(litEdge.fromEntityId);
            const to = placed.byId.get(litEdge.toEntityId);
            if (from !== undefined && to !== undefined) {
                const dash = litEdge.status === "proposed" ? [4, 3] : [];
                if (litEdge.kind === "parent") {
                    connector(context, { x: to.x + to.width, y: to.y + nodeHeight / 2 }, { x: from.x, y: from.y + nodeHeight / 2 }, colors.accent, 2, dash);
                } else {
                    arc(context, { x: from.x + from.width / 2, y: from.y + nodeHeight }, { x: to.x + to.width / 2, y: to.y + nodeHeight }, colors.accent, 2, dash);
                }
            }
        }

        for (const node of nodes) {
            const row = byRow.get(node.id);
            const size = sizes.get(node.id);
            if (row === undefined || size === undefined) {
                continue;
            }
            const dim = lit !== null && row.kind === "entity" && !lit.has(node.id);
            context.globalAlpha = dim ? 0.3 : 1;

            if (row.kind === "more") {
                context.setLineDash([3, 3]);
                roundRect(context, node.x, node.y, node.width, node.height, 6);
                context.strokeStyle = colors.edge;
                context.lineWidth = 1;
                context.stroke();
                context.setLineDash([]);
                if (detail === "full") {
                    context.fillStyle = colors.inkDim;
                    context.font = faces.faint;
                    context.textBaseline = "middle";
                    context.fillText(size.text, node.x + padding, node.y + nodeHeight / 2);
                }
                context.globalAlpha = 1;
                continue;
            }

            const held = graph.byId.get(node.id);
            const flags = graph.problems.get(node.id);
            const isSelected = node.id === selected;
            const isHovered = node.id === hovered;
            const isNeighbour = neighbours.has(node.id);
            const tone = toneColor(colors, kindTone(held?.kind ?? ""));
            const showLabel = labelled(detail, node.depth, row.childCount, isSelected);

            const grade: Severity | null = proof === null ? null : (proof.get(node.id) ?? "muted");
            const state = nodeStateOf(graph, node.id);
            roundRect(context, node.x, node.y, node.width, node.height, 6);
            if (!showLabel) {
                context.fillStyle = grade !== null ? proofStroke[grade] : state === "noPage" ? tone : stateStroke[state];
                context.globalAlpha = dim ? 0.15 : 0.4;
                context.fill();
                context.globalAlpha = 1;
            } else {
                context.fillStyle = grade !== null ? proofFill[grade] : isSelected ? colors.accentSoft : isHovered ? colors.raised : stateFill[state];
                context.fill();
            }
            if (glow > 0 && pulsing !== null && pulsing.ids.has(node.id)) {
                context.save();
                context.globalAlpha = glow * 0.9;
                context.strokeStyle = colors.accent;
                context.lineWidth = 3 + (1 - glow) * 6;
                roundRect(context, node.x - 3, node.y - 3, node.width + 6, node.height + 6, 9);
                context.stroke();
                context.restore();
                roundRect(context, node.x, node.y, node.width, node.height, 6);
            }
            const isLitEnd = litEnds.has(node.id);
            context.lineWidth = isSelected || isLitEnd ? 1.5 : 1;
            if (isLitEnd) {
                context.strokeStyle = colors.accent;
            } else if (grade !== null && !isSelected) {
                context.strokeStyle = proofStroke[grade];
            } else if (flags?.noPage) {
                context.strokeStyle = colors.danger;
                context.setLineDash([4, 3]);
            } else if (graph.placementProposed.has(node.id)) {
                context.strokeStyle = colors.info;
                context.setLineDash([4, 3]);
            } else if (state !== "noPage" && !isSelected && !isNeighbour) {
                context.strokeStyle = stateStroke[state];
            } else {
                context.strokeStyle = isSelected ? colors.accent : isNeighbour ? colors.info : colors.hairline;
            }
            context.stroke();
            context.setLineDash([]);

            if (showLabel) {
                if (held !== undefined) {
                    drawIcon(context, entityIcon(held.kind), node.x + padding, node.y + (nodeHeight - iconSize) / 2, iconSize, tone);
                }
                context.fillStyle = dim ? colors.inkDim : colors.ink;
                context.font = faces.label;
                context.textBaseline = "middle";
                context.fillText(size.text, node.x + padding + iconSize + 6, node.y + nodeHeight / 2 + 0.5);

                let cursor = node.x + node.width - padding - (size.chipWidth > 0 ? size.chipWidth + 6 : 0);
                if ((flags?.proposed ?? 0) > 0) {
                    cursor -= 22;
                    roundRect(context, cursor, node.y + 5, 18, 14, 4);
                    context.fillStyle = colors.infoSoft;
                    context.fill();
                    context.fillStyle = colors.info;
                    context.font = faces.chip;
                    context.textAlign = "center";
                    context.fillText(String(flags?.proposed ?? 0), cursor + 9, node.y + nodeHeight / 2 + 0.5);
                    context.textAlign = "left";
                }
                if (flags?.multiParent) {
                    cursor -= badgeSize + 4;
                    drawIcon(context, CallSplitIcon, cursor, node.y + (nodeHeight - badgeSize) / 2, badgeSize, colors.inkDim);
                }
                if (flags?.noPage) {
                    cursor -= badgeSize + 4;
                    drawIcon(context, LinkOffIcon, cursor, node.y + (nodeHeight - badgeSize) / 2, badgeSize, colors.danger);
                }
                if (size.chipWidth > 0) {
                    const chipX = node.x + node.width - padding - size.chipWidth;
                    roundRect(context, chipX, node.y + 4, size.chipWidth, nodeHeight - 8, 4);
                    context.fillStyle = row.expanded ? colors.inset : colors.raised;
                    context.fill();
                    chevron(context, chipX + 1, node.y + (nodeHeight - 12) / 2, 12, row.expanded, colors.inkSoft);
                    if (size.chip !== "") {
                        context.fillStyle = colors.inkSoft;
                        context.font = faces.chip;
                        context.fillText(size.chip, chipX + 14, node.y + nodeHeight / 2 + 0.5);
                    }
                }
            } else if (detail === "roots" && node.depth <= 1 && held !== undefined) {
                context.fillStyle = colors.inkSoft;
                context.font = `600 ${Math.round(12 / current.k)}px ${colors.fontSans}`;
                context.textBaseline = "bottom";
                context.fillText(held.name, node.x, node.y - 4 / current.k);
            }
            context.globalAlpha = 1;
        }

        const dragging = drag.current;
        if (dragging !== null && dragging.mode === "node" && dragging.moved && dragging.target !== null) {
            const ghost = dragging.target;
            const size = sizes.get(ghost.id);
            const held = graph.byId.get(ghost.id);
            if (dragging.over !== null) {
                const target = placed.byId.get(dragging.over);
                if (target !== undefined) {
                    context.setLineDash([4, 3]);
                    context.strokeStyle = colors.accent;
                    context.lineWidth = 2;
                    roundRect(context, target.x - 3, target.y - 3, target.width + 6, target.height + 6, 9);
                    context.stroke();
                    context.setLineDash([]);
                }
            }
            context.globalAlpha = 0.85;
            roundRect(context, dragging.cursor.x + 10, dragging.cursor.y + 10, ghost.width, ghost.height, 6);
            context.fillStyle = colors.raised;
            context.fill();
            context.strokeStyle = colors.accent;
            context.lineWidth = 1;
            context.stroke();
            if (size !== undefined && held !== undefined) {
                drawIcon(context, entityIcon(held.kind), dragging.cursor.x + 10 + padding, dragging.cursor.y + 10 + (nodeHeight - iconSize) / 2, iconSize, toneColor(colors, kindTone(held.kind)));
                context.fillStyle = colors.ink;
                context.font = faces.label;
                context.textBaseline = "middle";
                context.fillText(size.text, dragging.cursor.x + 10 + padding + iconSize + 6, dragging.cursor.y + 10 + nodeHeight / 2 + 0.5);
            }
            context.globalAlpha = 1;
        }
        context.restore();
    }, []);

    const worldOf = (pointer: Point): Point => toWorld(view.current ?? { x: 0, y: 0, k: 1 }, pointer);

    const droppable = (childId: string, overId: string): boolean => {
        const graph = latest.current.index;
        if (overId === childId || graph.placementParent.get(childId) === overId) {
            return false;
        }
        let current: string | undefined = overId;
        while (current !== undefined) {
            if (current === childId) {
                return false;
            }
            current = graph.placementParent.get(current);
        }
        return true;
    };

    const onPointerDown = (pointer: PointerInfo): void => {
        if (pointer.button !== 0 || view.current === null) {
            return;
        }
        const world = worldOf(pointer);
        const target = latest.current.grid.at(world.x, world.y);
        const row = target === null ? undefined : latest.current.rowById.get(target.id);
        drag.current = {
            start: pointer,
            view: view.current,
            moved: false,
            target,
            mode: row?.kind === "entity" ? "node" : "pan",
            cursor: world,
            over: null,
        };
    };

    const onPointerMove = (pointer: PointerInfo): void => {
        const dragging = drag.current;
        if (dragging !== null) {
            const dx = pointer.x - dragging.start.x;
            const dy = pointer.y - dragging.start.y;
            if (!dragging.moved && Math.hypot(dx, dy) < dragThreshold) {
                return;
            }
            dragging.moved = true;
            if (dragging.mode === "pan") {
                commitView(panBy(dragging.view, dx, dy));
                return;
            }
            const world = worldOf(pointer);
            dragging.cursor = world;
            const hit = latest.current.grid.at(world.x, world.y);
            const row = hit === null ? undefined : latest.current.rowById.get(hit.id);
            dragging.over = dragging.target !== null && row?.kind === "entity" && droppable(dragging.target.id, row.id) ? row.id : null;
            host.current?.redraw();
            return;
        }
        const world = worldOf(pointer);
        const hit = latest.current.grid.at(world.x, world.y);
        const next = hit?.id ?? null;
        if (next !== hover.current) {
            hover.current = next;
            host.current?.redraw();
            onHover?.(latest.current.rowById.get(next ?? "")?.kind === "entity" ? next : null, pointer);
        }
    };

    const onPointerUp = (pointer: PointerInfo): void => {
        const dragging = drag.current;
        drag.current = null;
        if (dragging === null) {
            return;
        }
        if (dragging.moved) {
            if (dragging.mode === "node") {
                host.current?.redraw();
                if (dragging.target !== null && dragging.over !== null) {
                    onReparent(dragging.target.id, dragging.over);
                }
            }
            return;
        }
        const target = dragging.target;
        if (target === null) {
            onSelect(null);
            return;
        }
        const row = latest.current.rowById.get(target.id);
        if (row === undefined) {
            return;
        }
        if (row.kind === "more") {
            onLiftMore(row.parentId, childLimit);
            return;
        }
        const size = latest.current.measures.get(target.id);
        const world = worldOf(pointer);
        if (size !== undefined && size.chipWidth > 0 && world.x >= target.x + target.width - padding - size.chipWidth) {
            onToggle(target.id);
            return;
        }
        onPick(target.id);
    };

    const onDoubleClick = (pointer: PointerInfo): void => {
        const world = worldOf(pointer);
        const hit = latest.current.grid.at(world.x, world.y);
        const row = hit === null ? undefined : latest.current.rowById.get(hit.id);
        if (row?.kind === "entity" && row.childCount > 0) {
            onToggle(row.id);
        }
    };

    const onWheel = useCallback(
        (wheel: WheelInfo): void => {
            const current = view.current;
            if (current === null) {
                return;
            }
            if (wheel.ctrlKey || wheel.metaKey) {
                commitView(zoomAt(current, wheel, Math.exp(-wheel.deltaY * 0.002)));
            } else if (wheel.shiftKey) {
                commitView(panBy(current, -wheel.deltaY, 0));
            } else {
                commitView(panBy(current, -wheel.deltaX, -wheel.deltaY));
            }
        },
        [commitView],
    );

    const onResize = useCallback(
        (next: Size): void => {
            const first = size.current.width === 0 && next.width > 0;
            size.current = next;
            if (first && view.current === null && latest.current.palette !== null && latest.current.layout.nodes.length > 0) {
                fitAll();
            }
        },
        [fitAll],
    );

    const navKeys: Readonly<Record<string, NavKey>> = {
        ArrowUp: "up",
        ArrowDown: "down",
        ArrowLeft: "left",
        ArrowRight: "right",
        Home: "home",
        End: "end",
    };

    const onKeyDown = (event: KeyboardEvent<HTMLCanvasElement>): void => {
        const nav = navKeys[event.key];
        if (nav !== undefined) {
            event.preventDefault();
            const next = move(latest.current.rows, latest.current.selectedId, nav);
            if (next === null) {
                return;
            }
            if (next.expand) {
                onToggle(next.id);
            } else {
                onSelect(next.id);
            }
            return;
        }
        const selected = latest.current.selectedId;
        if (event.key === " " && selected !== null) {
            event.preventDefault();
            onToggle(selected);
        } else if (event.key === "Escape") {
            if (drag.current !== null) {
                drag.current = null;
                host.current?.redraw();
                return;
            }
            onSelect(null);
        } else if (event.key === "0") {
            fitAll();
        } else if (event.key === "n" || event.key === "N") {
            onCreateChild(selected);
        } else if ((event.key === "e" || event.key === "E") && selected !== null) {
            onConnect(selected);
        } else if (event.key === "Delete" && selected !== null) {
            onDelete(selected);
        } else if (event.key === "+" || event.key === "=") {
            const current = view.current;
            if (current !== null) {
                commitView(zoomAt(current, { x: size.current.width / 2, y: size.current.height / 2 }, 1.2));
            }
        } else if (event.key === "-") {
            const current = view.current;
            if (current !== null) {
                commitView(zoomAt(current, { x: size.current.width / 2, y: size.current.height / 2 }, 1 / 1.2));
            }
        }
    };

    return (
        <div ref={frame} className="relative h-full min-h-0 w-full bg-canvas">
            <CanvasHost
                ref={host}
                label={copy.graph.label}
                draw={draw}
                onPointerDown={onPointerDown}
                onPointerMove={onPointerMove}
                onPointerUp={onPointerUp}
                onPointerLeave={() => {
                    if (hover.current !== null) {
                        hover.current = null;
                        host.current?.redraw();
                        onHover?.(null, { x: 0, y: 0 });
                    }
                }}
                onDoubleClick={onDoubleClick}
                onContextMenu={
                    onContextMenu === undefined
                        ? undefined
                        : (pointer) => {
                              const world = worldOf(pointer);
                              const hit = latest.current.grid.at(world.x, world.y);
                              const row = hit === null ? undefined : latest.current.rowById.get(hit.id);
                              onContextMenu(row?.kind === "entity" ? row.id : null, pointer);
                          }
                }
                onWheel={onWheel}
                onKeyDown={onKeyDown}
                onResize={onResize}
                cursor={drag.current?.moved ? "grabbing" : "default"}
            />
            {minimap && palette !== null && layout.nodes.length > 0 ? (
                <Minimap
                    siteId={siteId}
                    layout={layout}
                    palette={palette}
                    mapSize={() => size.current}
                    onJump={(world) => {
                        const current = view.current;
                        if (current !== null) {
                            commitView(centerOn(current, world, size.current));
                        }
                    }}
                />
            ) : null}
        </div>
    );
}
