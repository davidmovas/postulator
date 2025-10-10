"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreVertical, Pencil, Trash2, Unlink2 } from "lucide-react";
import { Topic } from "@/services/topic";

export interface TopicRowActionsProps {
  topic: Topic;
  disabled?: boolean;
  onEdit: (topic: Topic) => void;
  onUnassign: (topicId: number) => void | Promise<void>;
  onDelete: (topicId: number) => void | Promise<void>;
}

export function TopicRowActions({ topic, disabled = false, onEdit, onUnassign, onDelete }: TopicRowActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={disabled}>
          <MoreVertical className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onEdit(topic)}>
          <Pencil className="h-4 w-4 mr-2" />
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onUnassign(topic.id)}>
          <Unlink2 className="h-4 w-4 mr-2" />
          Unassign
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => onDelete(topic.id)}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default TopicRowActions;
