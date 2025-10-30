"use client";

import React, { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Topic, createTopic, updateTopic } from "@/services/topic";
import { DEFAULT_TOPIC_STRATEGY } from "@/constants/topics";
import type { TopicStrategy } from "@/constants/topics";
import { useErrorHandling } from "@/lib/error-handling";
import { assignTopicToSite, type Category } from "@/services/site";
import { CategorySelectWithSync } from "@/components/topics/fields/CategorySelectWithSync";
import { TopicStrategySelect } from "@/components/topics/fields/TopicStrategySelect";

export interface CreateEditTopicModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  topic?: Topic | null;
  siteId?: number; // optional context
  onSaved?: () => void | Promise<void>;
}

export function CreateEditTopicModal({ open, onOpenChange, topic, siteId, onSaved }: CreateEditTopicModalProps) {
  const { withErrorHandling } = useErrorHandling();

  const isEdit = !!topic;
  const [title, setTitle] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  // Category/Strategy (only used when creating within a site context)
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [strategy, setStrategy] = useState<TopicStrategy>(DEFAULT_TOPIC_STRATEGY);

  useEffect(() => {
    setTitle(topic?.title ?? "");
  }, [topic, open]);


  const handleSave = async () => {
    const trimmed = title.trim();
    if (!trimmed) return;
    setIsSaving(true);
    const ok = await withErrorHandling(async () => {
      if (isEdit && topic) {
        await updateTopic(topic.id, trimmed);
      } else {
        // Create the topic first
        const createdId = await createTopic(trimmed);
        // If we are in a site context, try to auto-assign the newly created topic to this site
        if (createdId != 0 && siteId && selectedCategory) {
          try {
              await assignTopicToSite(siteId, createdId, (selectedCategory.id), strategy);
          } catch (e) {
            // Silently ignore assignment errors; the topic is still created and can be assigned later
            console.warn("Auto-assign topic failed", e);
          }
        }
      }
    }, { successMessage: isEdit ? "Topic updated" : "Topic created", showSuccess: true });
    setIsSaving(false);
    if (ok !== null) {
      onOpenChange(false);
      if (onSaved) await onSaved();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit topic" : "Create topic"}</DialogTitle>
          <DialogDescription>
            {isEdit ? "Update the topic title." : "Create a new topic by providing a title."}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3 pt-2">
          <div className="space-y-2">
            <Label htmlFor="title">Title</Label>
            <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Enter topic title" />
          </div>

          {/* Category and strategy selection when creating within a site context */}
          {!isEdit && !!siteId && (
            <>
              <CategorySelectWithSync
                siteId={siteId}
                selectedCategory={selectedCategory}
                onChange={(c) => setSelectedCategory(c)}
              />

              <TopicStrategySelect
                value={strategy}
                onChange={setStrategy}
                disabled={isSaving}
              />
            </>
          )}
        </div>
        <DialogFooter className="pt-4">
          <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={isSaving}>Cancel</Button>
          <Button onClick={handleSave} disabled={!title.trim() || isSaving}>{isEdit ? "Save" : "Create"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default CreateEditTopicModal;
