import { useState, useCallback, useRef } from "react";
import { SitemapNode, CreateNodeInput } from "@/models/sitemaps";

// Types of actions that can be undone/redone
export type HistoryActionType =
    | "create_node"
    | "delete_node"
    | "update_node"
    | "move_node"        // Change parent
    | "move_position";   // Change canvas position

// Base action interface
interface BaseAction {
    type: HistoryActionType;
    timestamp: number;
}

// Create node action
interface CreateNodeAction extends BaseAction {
    type: "create_node";
    nodeId: number;
    nodeData: CreateNodeInput;
}

// Delete node action
interface DeleteNodeAction extends BaseAction {
    type: "delete_node";
    nodeId: number;
    nodeData: SitemapNode; // Full node data for restoration
}

// Update node action (title, slug, description, etc.)
interface UpdateNodeAction extends BaseAction {
    type: "update_node";
    nodeId: number;
    previousData: Partial<SitemapNode>;
    newData: Partial<SitemapNode>;
}

// Move node action (change parent)
interface MoveNodeAction extends BaseAction {
    type: "move_node";
    nodeId: number;
    previousParentId: number | null;
    newParentId: number | null;
}

// Move position action (canvas position)
interface MovePositionAction extends BaseAction {
    type: "move_position";
    nodeId: number;
    previousPosition: { x: number; y: number };
    newPosition: { x: number; y: number };
}

export type HistoryAction =
    | CreateNodeAction
    | DeleteNodeAction
    | UpdateNodeAction
    | MoveNodeAction
    | MovePositionAction;

interface UseSitemapHistoryOptions {
    maxHistory?: number;
}

// Service functions type for applying changes
export interface HistoryServiceFunctions {
    createNode: (input: CreateNodeInput) => Promise<SitemapNode | null>;
    createNodeWithPosition: (input: CreateNodeInput, x: number, y: number) => Promise<SitemapNode | null>;
    deleteNode: (nodeId: number) => Promise<void>;
    updateNode: (nodeId: number, data: Partial<SitemapNode>) => Promise<void>;
    moveNode: (nodeId: number, newParentId: number | undefined) => Promise<void>;
    updatePositions: (nodeId: number, x: number, y: number) => Promise<void>;
    reloadData: () => Promise<void>;
}

export function useSitemapHistory(options: UseSitemapHistoryOptions = {}) {
    const { maxHistory = 50 } = options;

    const [past, setPast] = useState<HistoryAction[]>([]);
    const [future, setFuture] = useState<HistoryAction[]>([]);
    const [isApplying, setIsApplying] = useState(false);

    // Service functions ref - set by the component
    const servicesRef = useRef<HistoryServiceFunctions | null>(null);

    const canUndo = past.length > 0 && !isApplying;
    const canRedo = future.length > 0 && !isApplying;

    // Set service functions
    const setServices = useCallback((services: HistoryServiceFunctions) => {
        servicesRef.current = services;
    }, []);

    // Record an action (call after making the change)
    const record = useCallback((action: Omit<HistoryAction, "timestamp">) => {
        if (isApplying) return; // Don't record during undo/redo

        const fullAction = { ...action, timestamp: Date.now() } as HistoryAction;

        setPast((prev) => {
            const newPast = [...prev, fullAction];
            if (newPast.length > maxHistory) {
                newPast.shift();
            }
            return newPast;
        });
        // Clear future when new action is recorded
        setFuture([]);
    }, [maxHistory, isApplying]);

    // Undo the last action
    const undo = useCallback(async () => {
        if (past.length === 0 || !servicesRef.current || isApplying) return;

        const services = servicesRef.current;
        const action = past[past.length - 1];

        setIsApplying(true);

        try {
            switch (action.type) {
                case "create_node":
                    // Undo create = delete the node
                    await services.deleteNode(action.nodeId);
                    break;

                case "delete_node": {
                    // Undo delete = recreate the node with its original position
                    const newNode = await services.createNodeWithPosition(
                        {
                            sitemapId: action.nodeData.sitemapId,
                            parentId: action.nodeData.parentId ?? undefined,
                            title: action.nodeData.title,
                            slug: action.nodeData.slug,
                            description: action.nodeData.description,
                            contentType: action.nodeData.contentType,
                            keywords: action.nodeData.keywords,
                        },
                        action.nodeData.positionX ?? 0,
                        action.nodeData.positionY ?? 0
                    );
                    // Update the action with new node ID for redo to work correctly
                    if (newNode) {
                        action.nodeId = newNode.id;
                        action.nodeData = { ...action.nodeData, id: newNode.id };
                    }
                    break;
                }

                case "update_node":
                    // Undo update = restore previous data
                    await services.updateNode(action.nodeId, action.previousData);
                    break;

                case "move_node":
                    // Undo move = restore previous parent
                    await services.moveNode(
                        action.nodeId,
                        action.previousParentId ?? undefined
                    );
                    break;

                case "move_position":
                    // Undo position change = restore previous position
                    await services.updatePositions(
                        action.nodeId,
                        action.previousPosition.x,
                        action.previousPosition.y
                    );
                    break;
            }

            // Move action to future
            setPast((prev) => prev.slice(0, -1));
            setFuture((prev) => [action, ...prev]);

            // Reload data to reflect changes
            await services.reloadData();
        } catch (error) {
            console.error("[History] Undo failed:", error);
        } finally {
            setIsApplying(false);
        }
    }, [past, isApplying]);

    // Redo the next action
    const redo = useCallback(async () => {
        if (future.length === 0 || !servicesRef.current || isApplying) return;

        const services = servicesRef.current;
        const action = future[0];

        setIsApplying(true);

        try {
            switch (action.type) {
                case "create_node": {
                    // Redo create = create the node again
                    const newNode = await services.createNode(action.nodeData);
                    // Update the action with new node ID for future undo to work
                    if (newNode) {
                        action.nodeId = newNode.id;
                    }
                    break;
                }

                case "delete_node":
                    // Redo delete = delete the node again
                    await services.deleteNode(action.nodeId);
                    break;

                case "update_node":
                    // Redo update = apply new data
                    await services.updateNode(action.nodeId, action.newData);
                    break;

                case "move_node":
                    // Redo move = apply new parent
                    await services.moveNode(
                        action.nodeId,
                        action.newParentId ?? undefined
                    );
                    break;

                case "move_position":
                    // Redo position change = apply new position
                    await services.updatePositions(
                        action.nodeId,
                        action.newPosition.x,
                        action.newPosition.y
                    );
                    break;
            }

            // Move action to past
            setFuture((prev) => prev.slice(1));
            setPast((prev) => [...prev, action]);

            // Reload data to reflect changes
            await services.reloadData();
        } catch (error) {
            console.error("[History] Redo failed:", error);
        } finally {
            setIsApplying(false);
        }
    }, [future, isApplying]);

    // Clear all history
    const clear = useCallback(() => {
        setPast([]);
        setFuture([]);
    }, []);

    // Helper functions to record specific actions
    const recordCreateNode = useCallback((nodeId: number, nodeData: CreateNodeInput) => {
        record({ type: "create_node", nodeId, nodeData });
    }, [record]);

    const recordDeleteNode = useCallback((nodeData: SitemapNode) => {
        record({ type: "delete_node", nodeId: nodeData.id, nodeData });
    }, [record]);

    const recordUpdateNode = useCallback((
        nodeId: number,
        previousData: Partial<SitemapNode>,
        newData: Partial<SitemapNode>
    ) => {
        record({ type: "update_node", nodeId, previousData, newData });
    }, [record]);

    const recordMoveNode = useCallback((
        nodeId: number,
        previousParentId: number | null,
        newParentId: number | null
    ) => {
        record({ type: "move_node", nodeId, previousParentId, newParentId });
    }, [record]);

    const recordMovePosition = useCallback((
        nodeId: number,
        previousPosition: { x: number; y: number },
        newPosition: { x: number; y: number }
    ) => {
        record({ type: "move_position", nodeId, previousPosition, newPosition });
    }, [record]);

    return {
        // State
        canUndo,
        canRedo,
        isApplying,
        historyLength: past.length,
        futureLength: future.length,

        // Setup
        setServices,

        // Actions
        undo,
        redo,
        clear,

        // Record helpers
        recordCreateNode,
        recordDeleteNode,
        recordUpdateNode,
        recordMoveNode,
        recordMovePosition,
    };
}
