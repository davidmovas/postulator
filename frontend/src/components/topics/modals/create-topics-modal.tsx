"use client";

import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useApiCall } from "@/hooks/use-api-call";
import { topicService } from "@/services/topics";
import { useToast } from "@/components/ui/use-toast";
import { TopicCreateInput, BatchResult } from "@/models/topics";

interface CreateTopicsModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number;
    onSuccess?: () => void;
}

export function CreateTopicsModal({ open, onOpenChange, siteId, onSuccess }: CreateTopicsModalProps) {
    const { execute, isLoading } = useApiCall();
    const { toast } = useToast();

    const [value, setValue] = useState<string>("");

    const titles = useMemo(() => {
        return value
            .split(/\r?\n/)
            .map(s => s.trim())
            .filter(s => s.length > 0);
    }, [value]);

    const isFormValid = titles.length > 0;

    const resetForm = () => setValue("");

    const handleSubmit = async () => {
        if (!isFormValid) return;

        const payload: TopicCreateInput[] = titles.map(t => ({ title: t }));

        const result = await execute<BatchResult>(() => topicService.createTopics(payload), {
            errorTitle: "Failed to create topics",
            showSuccessToast: false,
        });

        if (result) {
            // Assign newly created topics to the site if any were created
            const createdIds = (result.createdTopics || []).map(t => t.id);
            if (createdIds.length > 0) {
                await execute(
                    () => topicService.assignToSite(siteId, createdIds),
                    {
                        errorTitle: "Failed to assign topics to site",
                        showSuccessToast: false,
                    }
                );
            }

            const description = `Created: ${result.created} | Skipped: ${result.skipped}`;

            toast({
                title: "Topics creation result",
                description,
                variant: "success",
            });

            onOpenChange(false);
            resetForm();
            onSuccess?.();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            resetForm();
        }
        onOpenChange(newOpen);
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[600px]">
                <DialogHeader>
                    <DialogTitle>Add Topics</DialogTitle>
                    <DialogDescription>
                        Enter one topic title per line. Each line will be created as a separate topic.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-3 py-2">
                    <div className="space-y-2">
                        <Label htmlFor="topics">Titles</Label>
                        <Textarea
                            id="topics"
                            placeholder={"One topic per line"}
                            value={value}
                            onChange={(e) => setValue(e.target.value)}
                            rows={10}
                            disabled={isLoading}
                        />
                        <div className="text-xs text-muted-foreground">{titles.length} title{titles.length === 1 ? "" : "s"} detected</div>
                    </div>
                </div>

                <DialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => handleOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Creating..." : "Create Topics"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
