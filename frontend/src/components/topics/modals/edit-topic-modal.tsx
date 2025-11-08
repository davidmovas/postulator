"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useApiCall } from "@/hooks/use-api-call";
import { topicService } from "@/services/topics";
import { Topic, TopicUpdateInput } from "@/models/topics";

interface EditTopicModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    topic: Topic | null;
    onSuccess?: () => void;
}

export function EditTopicModal({ open, onOpenChange, topic, onSuccess }: EditTopicModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<TopicUpdateInput>({
        id: 0,
        title: "",
    });

    useEffect(() => {
        if (topic) {
            setFormData({
                id: topic.id,
                title: topic.title,
            });
        }
    }, [topic]);

    const isFormValid = !!formData.title && formData.title.trim().length > 0;

    const handleSubmit = async () => {
        if (!isFormValid || !topic) return;

        const result = await execute<void>(
            () => topicService.updateTopic({ id: formData.id, title: formData.title!.trim() }),
            {
                successMessage: "Topic updated successfully",
                showSuccessToast: true,
                errorTitle: "Failed to update topic",
            }
        );

        if (result !== null) {
            onOpenChange(false);
            onSuccess?.();
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Edit Topic</DialogTitle>
                    <DialogDescription>
                        Update the topic title.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="topic-title">Title<span className="text-red-600">*</span></Label>
                        <Input
                            id="topic-title"
                            value={formData.title || ""}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                title: e.target.value,
                            }))}
                            disabled={isLoading}
                            placeholder="Enter topic title"
                        />
                    </div>
                </div>

                <DialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Updating..." : "Update Topic"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
