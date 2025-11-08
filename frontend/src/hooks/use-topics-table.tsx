"use client";

import { useState, useMemo, useCallback } from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RiPencilLine } from "@remixicon/react";
import { Topic } from "@/models/topics";
import { topicService } from "@/services/topics";
import { useApiCall } from "@/hooks/use-api-call";
import { useContextModal } from "@/context/modal-context";

export function useTopicsTable(siteId: number) {
    const [topics, setTopics] = useState<Topic[]>([]);
    const { execute, isLoading } = useApiCall();
    const { confirmationModal, editTopicModal } = useContextModal();

    const loadTopics = useCallback(async () => {
        const result = await execute<Topic[]>(
            () => topicService.getSiteTopics(siteId),
            { errorTitle: "Failed to load topics" }
        );
        if (result) setTopics(result);
    }, [siteId, execute]);

    const columns: ColumnDef<Topic>[] = useMemo(() => [
        {
            id: "select",
            header: ({ table }) => {
                const isAllSelected = table.getIsAllPageRowsSelected();
                return (
                    <input
                        type="checkbox"
                        aria-label="Select all"
                        checked={isAllSelected}
                        onChange={(e) => table.toggleAllPageRowsSelected(e.currentTarget.checked)}
                    />
                );
            },
            cell: ({ row }) => (
                <input
                    type="checkbox"
                    aria-label="Select row"
                    checked={row.getIsSelected()}
                    onChange={(e) => row.toggleSelected(e.currentTarget.checked)}
                />
            ),
            enableSorting: false,
            enableHiding: false,
            size: 32,
        },
        {
            accessorKey: "title",
            header: "Title",
            cell: ({ row }) => {
                const t = row.original;
                return <div className="font-medium break-words">{t.title}</div>;
            },
        },
        {
            accessorKey: "createdAt",
            header: "Created",
            cell: ({ row }) => {
                const created = row.getValue("createdAt") as string;
                const date = created ? new Date(created) : undefined;
                return <span className="text-muted-foreground">{date ? date.toLocaleDateString() : "-"}</span>;
            },
        },
        {
            id: "actions",
            header: "Actions",
            cell: ({ row }) => {
                const topic = row.original;

                const handleEdit = () => {
                    editTopicModal.open(topic);
                };

                const handleDelete = () => {
                    confirmationModal.open({
                        title: "Delete Topic",
                        description: (
                            <div>
                                Are you sure you want to delete this topic?
                                <div className="mt-2 p-2 rounded bg-muted/50 text-sm">
                                    {topic.title}
                                </div>
                            </div>
                        ),
                        confirmText: "Delete",
                        variant: "destructive",
                        onConfirm: async () => {
                            await topicService.deleteTopic(topic.id);
                            setTopics(prev => prev.filter(t => t.id !== topic.id));
                        }
                    });
                };

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>

                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={handleEdit}>
                                <RiPencilLine className="mr-2 h-4 w-4" />
                                <span>Edit</span>
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem onClick={handleDelete} className="text-red-600">
                                <Trash2 className="mr-2 h-4 w-4" />
                                <span>Delete</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                );
            },
        },
    ], [editTopicModal, confirmationModal]);

    return {
        topics,
        setTopics,
        columns,
        isLoading,
        loadTopics,
    };
}
