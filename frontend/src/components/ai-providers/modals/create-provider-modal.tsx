"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { ProviderCreateInput } from "@/models/providers";
import { useApiCall } from "@/hooks/use-api-call";
import { useProviderModels } from "@/hooks/use-provider-models";
import { providerService } from "@/services/providers";
import { Cpu, DollarSign, Hash } from "lucide-react";
import { PROVIDER_TYPES } from "@/constants/providers";

interface CreateProviderModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess?: () => void;
}

export function CreateProviderModal({ open, onOpenChange, onSuccess }: CreateProviderModalProps) {
    const { execute, isLoading } = useApiCall();
    const { availableModels, modelsLoading, loadModels } = useProviderModels();
    const [selectedModelInfo, setSelectedModelInfo] = useState<any>(null);

    const [formData, setFormData] = useState<ProviderCreateInput>({
        name: "",
        type: "",
        apiKey: "",
        model: "",
        isActive: true
    });

    const resetForm = () => {
        setFormData({
            name: "",
            type: "",
            apiKey: "",
            model: "",
            isActive: true
        });
        setSelectedModelInfo(null);
    };

    const handleTypeChange = (type: string) => {
        setFormData(prev => ({
            ...prev,
            type,
            model: "" // Reset model when type changes
        }));
        setSelectedModelInfo(null);
        loadModels(type);
    };

    const handleModelChange = (modelId: string) => {
        setFormData(prev => ({ ...prev, model: modelId }));

        const modelInfo = availableModels.find(m => m.id === modelId);
        setSelectedModelInfo(modelInfo);
    };

    const isFormValid = formData.name.trim() &&
        formData.type &&
        formData.apiKey.trim() &&
        formData.model;

    const handleSubmit = async () => {
        if (!isFormValid) return;

        const result = await execute<string>(
            () => providerService.createProvider(formData),
            {
                successMessage: "Provider created successfully",
                showSuccessToast: true
            }
        );

        if (result) {
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

    const formatCost = (cost: number): string => {
        return `$${cost.toFixed(2)}`;
    };

    const formatTokens = (tokens: number): string => {
        if (tokens >= 1000000) {
            return `${(tokens / 1000000).toFixed(1)}M`;
        }
        if (tokens >= 1000) {
            return `${(tokens / 1000).toFixed(1)}K`;
        }
        return tokens.toString();
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Create AI Provider</DialogTitle>
                    <DialogDescription>
                        Add a new AI provider to use with your content generation.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="name">Provider Name<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="name"
                            placeholder="My OpenAI Provider"
                            value={formData.name}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                name: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="type">Provider Type<span className="text-lg text-red-600">*</span></Label>
                        <Select value={formData.type} onValueChange={handleTypeChange} disabled={isLoading}>
                            <SelectTrigger>
                                <SelectValue placeholder="Select provider type" />
                            </SelectTrigger>
                            <SelectContent>
                                {Object.entries(PROVIDER_TYPES).map(([key, config]) => (
                                    <SelectItem key={key} value={key}>
                                        {config.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="model">Model<span className="text-lg text-red-600">*</span></Label>
                        <Select
                            value={formData.model}
                            onValueChange={handleModelChange}
                            disabled={isLoading || !formData.type || modelsLoading}
                        >
                            <SelectTrigger>
                                <SelectValue placeholder={
                                    modelsLoading ? "Loading models..." :
                                        !formData.type ? "Select provider type first" :
                                            "Select model"
                                } />
                            </SelectTrigger>
                            <SelectContent>
                                {availableModels.map(model => (
                                    <SelectItem key={model.id} value={model.id}>
                                        {model.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Model Information Card */}
                    {selectedModelInfo && (
                        <div className="bg-muted/40 border rounded-lg p-3 space-y-2">
                            <div className="flex items-center gap-2">
                                <Cpu className="h-4 w-4 text-blue-500" />
                                <h4 className="font-medium text-sm">Model Specifications</h4>
                            </div>

                            <div className="grid grid-cols-2 gap-4 text-sm">
                                <div className="space-y-1">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <Hash className="h-3 w-3" />
                                        <span>Context Window</span>
                                    </div>
                                    <Badge variant="outline" className="font-mono text-xs">
                                        {formatTokens(selectedModelInfo.contextWindow)}
                                    </Badge>
                                </div>

                                <div className="space-y-1">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <Hash className="h-3 w-3" />
                                        <span>Max Output</span>
                                    </div>
                                    <Badge variant="outline" className="font-mono text-xs">
                                        {formatTokens(selectedModelInfo.maxOutputTokens)}
                                    </Badge>
                                </div>

                                <div className="space-y-1">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <DollarSign className="h-3 w-3" />
                                        <span>Input per 1M</span>
                                    </div>
                                    <Badge variant="outline" className="font-mono text-xs">
                                        {formatCost(selectedModelInfo.inputCost)}
                                    </Badge>
                                </div>

                                <div className="space-y-1">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <DollarSign className="h-3 w-3" />
                                        <span>Output per 1M</span>
                                    </div>
                                    <Badge variant="outline" className="font-mono text-xs">
                                        {formatCost(selectedModelInfo.outputCost)}
                                    </Badge>
                                </div>
                            </div>

                        </div>
                    )}

                    <div className="space-y-2">
                        <Label htmlFor="apiKey">API Key<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="apiKey"
                            type="password"
                            placeholder="Enter your API key"
                            value={formData.apiKey}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                apiKey: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="flex items-center justify-between">
                        <Label htmlFor="isActive" className="flex flex-col space-y-1">
                            <span>Active</span>
                            <span className="font-normal text-sm text-muted-foreground">
                                Enable this provider for content generation
                            </span>
                        </Label>
                        <Switch
                            id="isActive"
                            checked={formData.isActive}
                            onCheckedChange={(checked) => setFormData(prev => ({
                                ...prev,
                                isActive: checked
                            }))}
                            disabled={isLoading}
                        />
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
                        {isLoading ? "Creating..." : "Create Provider"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}