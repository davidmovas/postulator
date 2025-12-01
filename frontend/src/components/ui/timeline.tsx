"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

interface TimelineProps extends React.HTMLAttributes<HTMLDivElement> {
    children: React.ReactNode;
}

const Timeline = React.forwardRef<HTMLDivElement, TimelineProps>(
    ({ className, children, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn("relative", className)}
                {...props}
            >
                {children}
            </div>
        );
    }
);
Timeline.displayName = "Timeline";

interface TimelineItemProps extends React.HTMLAttributes<HTMLDivElement> {
    isLast?: boolean;
    isActive?: boolean;
    isCompleted?: boolean;
}

const TimelineItem = React.forwardRef<HTMLDivElement, TimelineItemProps>(
    ({ className, children, isLast, isActive, isCompleted, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn(
                    "relative pl-8 pb-4",
                    isLast && "pb-0",
                    className
                )}
                data-active={isActive}
                data-completed={isCompleted}
                {...props}
            >
                {children}
            </div>
        );
    }
);
TimelineItem.displayName = "TimelineItem";

interface TimelineIndicatorProps extends React.HTMLAttributes<HTMLDivElement> {
    isActive?: boolean;
    isCompleted?: boolean;
    isError?: boolean;
}

const TimelineIndicator = React.forwardRef<HTMLDivElement, TimelineIndicatorProps>(
    ({ className, children, isActive, isCompleted, isError, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn(
                    "absolute left-0 top-3 flex h-6 w-6 items-center justify-center rounded-full border-2 bg-background z-10",
                    isCompleted && "border-green-500 bg-green-500 text-white",
                    isActive && "border-primary bg-primary text-primary-foreground",
                    isError && "border-destructive bg-destructive text-white",
                    !isCompleted && !isActive && !isError && "border-muted-foreground/30 text-muted-foreground",
                    className
                )}
                {...props}
            >
                {children}
            </div>
        );
    }
);
TimelineIndicator.displayName = "TimelineIndicator";

interface TimelineSeparatorProps extends React.HTMLAttributes<HTMLDivElement> {
    isCompleted?: boolean;
    isLast?: boolean;
}

const TimelineSeparator = React.forwardRef<HTMLDivElement, TimelineSeparatorProps>(
    ({ className, isCompleted, isLast, ...props }, ref) => {
        if (isLast) return null;
        return (
            <div
                ref={ref}
                className={cn(
                    "absolute left-[11px] top-9 bottom-[-1rem] w-0.5",
                    isCompleted ? "bg-green-500" : "bg-muted-foreground/20",
                    className
                )}
                {...props}
            />
        );
    }
);
TimelineSeparator.displayName = "TimelineSeparator";

interface TimelineContentProps extends React.HTMLAttributes<HTMLDivElement> {}

const TimelineContent = React.forwardRef<HTMLDivElement, TimelineContentProps>(
    ({ className, children, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn("pt-0.5", className)}
                {...props}
            >
                {children}
            </div>
        );
    }
);
TimelineContent.displayName = "TimelineContent";

interface TimelineHeaderProps extends React.HTMLAttributes<HTMLDivElement> {}

const TimelineHeader = React.forwardRef<HTMLDivElement, TimelineHeaderProps>(
    ({ className, children, ...props }, ref) => {
        return (
            <div
                ref={ref}
                className={cn("flex items-center gap-2 font-medium", className)}
                {...props}
            >
                {children}
            </div>
        );
    }
);
TimelineHeader.displayName = "TimelineHeader";

export {
    Timeline,
    TimelineItem,
    TimelineIndicator,
    TimelineSeparator,
    TimelineContent,
    TimelineHeader,
};
