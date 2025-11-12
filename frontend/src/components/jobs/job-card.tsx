"use client";

import { Job } from "@/models/jobs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem, DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    Play,
    Pause,
    MoreVertical,
    Edit,
    Trash2,
    Calendar,
    Zap,
    CheckCircle,
    XCircle,
    Clock
} from "lucide-react";
import { formatDateTime } from "@/lib/time";

interface JobCardProps {
    job: Job;
    onEdit: (job: Job) => void;
    onDelete: (job: Job) => void;
    onPause: (job: Job) => void;
    onResume: (job: Job) => void;
    onExecute: (job: Job) => void;
}

export function JobCard({ job, onEdit, onDelete, onPause, onResume, onExecute }: JobCardProps) {
    const isActive = job.status === "active";
    const isPaused = job.status === "paused";

    const getScheduleInfo = () => {
        if (!job.schedule) return "Manual";

        switch (job.schedule.type) {
            case "once":
                return `Once at ${formatDateTime(job.schedule.config.executeAt)}`;
            case "interval":
                return `Every ${job.schedule.config.value} ${job.schedule.config.unit}`;
            case "daily":
                return `Daily at ${job.schedule.config.hour}:${job.schedule.config.minute.toString().padStart(2, '0')}`;
            default:
                return job.schedule.type;
        }
    };

    const getStatusIcon = () => {
        switch (job.status) {
            case "active": return <CheckCircle className="h-3 w-3 text-green-500" />;
            case "paused": return <Pause className="h-3 w-3 text-yellow-500" />;
            case "failed": return <XCircle className="h-3 w-3 text-red-500" />;
            default: return <Clock className="h-3 w-3 text-gray-500" />;
        }
    };

    return (
        <Card className="hover:shadow-lg transition-shadow duration-200">
            <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                    <div className="space-y-2">
                        <CardTitle className="text-lg font-bold line-clamp-2">
                            {job.name}
                        </CardTitle>
                        <div className="flex items-center gap-2">
                            <Badge variant={isActive ? "default" : "secondary"} className="flex items-center gap-1">
                                {getStatusIcon()}
                                {job.status}
                            </Badge>
                            {job.requiresValidation && (
                                <Badge variant="outline" className="text-xs">
                                    Requires Validation
                                </Badge>
                            )}
                            {job.jitterEnabled && (
                                <Badge variant="outline" className="text-xs">
                                    Jitter: {job.jitterMinutes}m
                                </Badge>
                            )}
                        </div>
                    </div>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                <MoreVertical className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            {isActive ? (
                                <DropdownMenuItem onClick={() => onPause(job)}>
                                    <Pause className="h-4 w-4 mr-2" />
                                    Pause
                                </DropdownMenuItem>
                            ) : (
                                <DropdownMenuItem onClick={() => onResume(job)}>
                                    <Play className="h-4 w-4 mr-2" />
                                    Resume
                                </DropdownMenuItem>
                            )}
                            <DropdownMenuItem onClick={() => onExecute(job)}>
                                <Zap className="h-4 w-4 mr-2" />
                                Run Now
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => onEdit(job)}>
                                <Edit className="h-4 w-4 mr-2" />
                                Edit
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem
                                onClick={() => onDelete(job)}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </CardHeader>

            <CardContent className="space-y-3">
                {/* Schedule Info */}
                <div className="flex items-center gap-2 text-sm">
                    <Calendar className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Schedule:</span>
                    <span className="font-medium">{getScheduleInfo()}</span>
                </div>

                {/* Strategies */}
                <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                        <span className="text-muted-foreground">Topics:</span>
                        <Badge variant="outline" className="ml-2 text-xs">
                            {job.topicStrategy}
                        </Badge>
                    </div>
                    <div>
                        <span className="text-muted-foreground">Categories:</span>
                        <Badge variant="outline" className="ml-2 text-xs">
                            {job.categoryStrategy}
                        </Badge>
                    </div>
                </div>

                {/* Stats */}
                {job.state && (
                    <div className="grid grid-cols-3 gap-2 text-xs">
                        <div className="text-center">
                            <div className="font-semibold">{job.state.totalExecutions}</div>
                            <div className="text-muted-foreground">Total</div>
                        </div>
                        <div className="text-center">
                            <div className="font-semibold text-red-500">{job.state.failedExecutions}</div>
                            <div className="text-muted-foreground">Failed</div>
                        </div>
                        <div className="text-center">
                            <div className="font-semibold">
                                {job.state.lastRunAt ? formatDateTime(job.state.lastRunAt) : 'Never'}
                            </div>
                            <div className="text-muted-foreground">Last Run</div>
                        </div>
                    </div>
                )}

                {/* Next Run */}
                {job.state?.nextRunAt && isActive && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        Next: {formatDateTime(job.state.nextRunAt)}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}