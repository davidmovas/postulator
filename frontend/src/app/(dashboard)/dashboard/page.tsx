"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { RefreshCw, Globe, PlayCircle, PauseCircle, AlertTriangle, CheckCircle2, Clock, XCircle } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useDashboard } from "@/hooks/use-dashboard";
import { StatsCard } from "@/components/dashboard/stats-card";
import { LastUpdated } from "@/components/dashboard/last-updated";

const DashboardIcons = {
    sites: Globe,
    activeJobs: PlayCircle,
    pausedJobs: PauseCircle,
    unhealthy: AlertTriangle,
    healthy: CheckCircle2,
    pending: Clock,
    failed: XCircle
};

const borderColors = {
    sites: "border-l-blue-500",
    jobs: "border-l-purple-500",
    active: "border-l-green-500",
    warning: "border-l-orange-500",
    danger: "border-l-red-500",
    neutral: "border-l-gray-500"
};

export default function DashboardPage() {
    const [autoRefresh, setAutoRefresh] = useState(true);
    const { data, isLoading, error, lastUpdated, refresh } = useDashboard(autoRefresh);

    if (error) {
        return (
            <div className="flex items-center justify-center">
                <div className="text-center space-y-4">
                    <AlertTriangle className="h-16 w-16 text-destructive mx-auto" />
                    <h2 className="text-2xl font-bold text-destructive">Failed to load dashboard</h2>
                    <Button onClick={refresh} variant="outline">
                        Try Again
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="p-6 space-y-8">

            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
                    <p className="text-muted-foreground mt-1">
                        Overview of your sites and jobs performance
                    </p>
                </div>

                <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                        <Switch
                            checked={autoRefresh}
                            onCheckedChange={setAutoRefresh}
                            id="auto-refresh"
                        />
                        <Label htmlFor="auto-refresh" className="text-sm">
                            Auto-refresh
                        </Label>
                    </div>

                    <Button
                        onClick={refresh}
                        variant="outline"
                        size="sm"
                        disabled={isLoading}
                        className="flex items-center gap-2"
                    >
                        <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
                        Refresh
                    </Button>
                </div>
            </div>

            {/* Last Updated */}
            <div className="flex justify-end">
                <LastUpdated
                    lastUpdated={lastUpdated}
                    isLoading={isLoading}
                    autoRefresh={autoRefresh}
                />
            </div>

            {/* Main Stats Grid */}
            <AnimatePresence>
                {!isLoading && data && (
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="space-y-8"
                    >
                        {/* Sites Section */}
                        <div>
                            <h2 className="text-xl font-semibold mb-4">Sites Overview</h2>
                            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                                <StatsCard
                                    title="Total Sites"
                                    value={data.totalSites}
                                    icon={<DashboardIcons.sites />}
                                    description="All WordPress sites"
                                    className={borderColors.sites}
                                />

                                <StatsCard
                                    title="Active Sites"
                                    value={data.activeSites}
                                    icon={<DashboardIcons.healthy />}
                                    description="Currently operational"
                                    className={borderColors.active}
                                />

                                <StatsCard
                                    title="Unhealthy Sites"
                                    value={data.unhealthySites}
                                    icon={<DashboardIcons.unhealthy />}
                                    description="Requiring attention"
                                    className={borderColors.danger}
                                />
                            </div>
                        </div>

                        {/* Jobs Section */}
                        <div>
                            <h2 className="text-xl font-semibold mb-4">Jobs Performance</h2>
                            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                                <StatsCard
                                    title="Total Jobs"
                                    value={data.totalJobs}
                                    icon={<DashboardIcons.sites />}
                                    description="All configured jobs"
                                    className={borderColors.jobs}
                                />

                                <StatsCard
                                    title="Active Jobs"
                                    value={data.activeJobs}
                                    icon={<DashboardIcons.activeJobs />}
                                    description="Currently running"
                                    className={borderColors.active}
                                />

                                <StatsCard
                                    title="Paused Jobs"
                                    value={data.pausedJobs}
                                    icon={<DashboardIcons.pausedJobs />}
                                    description="Temporarily stopped"
                                    className={borderColors.warning}
                                />

                                <StatsCard
                                    title="Pending Validations"
                                    value={data.pendingValidations}
                                    icon={<DashboardIcons.pending />}
                                    description="Awaiting approval"
                                    className={borderColors.neutral}
                                />
                            </div>
                        </div>

                        {/* Activity Section */}
                        <div>
                            <h2 className="text-xl font-semibold mb-4">Today&apos;s Activity</h2>
                            <div className="grid gap-4 md:grid-cols-2">
                                <StatsCard
                                    title="Executions"
                                    value={data.executionsToday}
                                    icon={<DashboardIcons.healthy />}
                                    description="Successful job runs"
                                    className={borderColors.active}
                                />

                                <StatsCard
                                    title="Failed Executions"
                                    value={data.failedExecutionsToday}
                                    icon={<DashboardIcons.failed />}
                                    description="Unsuccessful runs"
                                    className={borderColors.danger}
                                />
                            </div>
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>

            {/* Loading State */}
            {isLoading && !data && (
                <div className="space-y-8">
                    {[...Array(3)].map((_, sectionIndex) => (
                        <div key={sectionIndex} className="space-y-4">
                            <div className="h-6 bg-muted/50 rounded-lg w-48 animate-pulse" />
                            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                                {[...Array(sectionIndex === 1 ? 4 : 3)].map((_, cardIndex) => (
                                    <div
                                        key={cardIndex}
                                        className="h-32 bg-muted/30 rounded-lg animate-pulse"
                                        style={{
                                            animationDelay: `${cardIndex * 0.1}s`
                                        }}
                                    />
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}