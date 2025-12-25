"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
    RefreshCw,
    Globe,
    PlayCircle,
    AlertTriangle,
    CheckCircle2,
    Clock,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useDashboard } from "@/hooks/use-dashboard";
import { LastUpdated } from "@/components/dashboard/last-updated";
import { DashboardCharts } from "@/components/dashboard/dashboard-charts";
import { AIUsageSection } from "@/components/dashboard/ai-usage-section";
import Link from "next/link";

export default function DashboardPage() {
    const { data, isLoading, error, lastUpdated, refresh } = useDashboard();

    if (error) {
        return (
            <div className="flex items-center justify-center min-h-[50vh]">
                <div className="text-center space-y-4">
                    <AlertTriangle className="h-16 w-16 text-destructive mx-auto" />
                    <h2 className="text-2xl font-bold text-destructive">
                        Failed to load dashboard
                    </h2>
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
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
                    <p className="text-muted-foreground text-sm">
                        Overview of your sites and jobs
                    </p>
                </div>

                <div className="flex items-center gap-4">
                    <LastUpdated
                        lastUpdated={lastUpdated}
                        isLoading={isLoading}
                        autoRefresh={true}
                    />
                    <Button
                        onClick={refresh}
                        variant="outline"
                        size="sm"
                        disabled={isLoading}
                    >
                        <RefreshCw
                            className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`}
                        />
                        Refresh
                    </Button>
                </div>
            </div>

            {/* Main Content */}
            <AnimatePresence>
                {!isLoading && data && (
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="space-y-8"
                    >
                        {/* Quick Stats Row - 4 cards */}
                        <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
                            <Link href="/sites/">
                                <Card className="border-l-4 border-l-blue-500 hover:bg-muted/50 transition-colors cursor-pointer h-full">
                                    <CardContent className="pt-4 pb-4">
                                        <div className="flex items-center gap-3">
                                            <div className="p-2 rounded-lg bg-blue-500/10">
                                                <Globe className="w-5 h-5 text-blue-500" />
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Sites</p>
                                                <p className="text-2xl font-bold">{data.totalSites}</p>
                                                <p className="text-xs text-muted-foreground">
                                                    {data.activeSites} active
                                                </p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            </Link>

                            <Link href="/sites/">
                                <Card className={`border-l-4 ${data.unhealthySites > 0 ? "border-l-yellow-500" : "border-l-green-500"} hover:bg-muted/50 transition-colors cursor-pointer h-full`}>
                                    <CardContent className="pt-4 pb-4">
                                        <div className="flex items-center gap-3">
                                            <div className={`p-2 rounded-lg ${data.unhealthySites > 0 ? "bg-yellow-500/10" : "bg-green-500/10"}`}>
                                                {data.unhealthySites > 0 ? (
                                                    <AlertTriangle className="w-5 h-5 text-yellow-500" />
                                                ) : (
                                                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                                                )}
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Health</p>
                                                <p className="text-2xl font-bold">
                                                    <span className="text-green-500">{data.totalSites - data.unhealthySites}</span>
                                                    <span className="text-muted-foreground mx-1">/</span>
                                                    <span className={data.unhealthySites > 0 ? "text-red-500" : "text-green-500"}>{data.unhealthySites}</span>
                                                </p>
                                                <p className="text-xs text-muted-foreground">
                                                    healthy / unhealthy
                                                </p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            </Link>

                            <Link href="/jobs/">
                                <Card className="border-l-4 border-l-purple-500 hover:bg-muted/50 transition-colors cursor-pointer h-full">
                                    <CardContent className="pt-4 pb-4">
                                        <div className="flex items-center gap-3">
                                            <div className="p-2 rounded-lg bg-purple-500/10">
                                                <PlayCircle className="w-5 h-5 text-purple-500" />
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Jobs</p>
                                                <p className="text-2xl font-bold">{data.totalJobs}</p>
                                                <p className="text-xs text-muted-foreground">
                                                    {data.activeJobs} active{data.pausedJobs > 0 ? `, ${data.pausedJobs} paused` : ""}
                                                </p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            </Link>

                            <Card className="border-l-4 border-l-cyan-500 h-full">
                                <CardContent className="pt-4 pb-4">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2 rounded-lg bg-cyan-500/10">
                                            <CheckCircle2 className="w-5 h-5 text-cyan-500" />
                                        </div>
                                        <div>
                                            <p className="text-xs text-muted-foreground">Today</p>
                                            <p className="text-2xl font-bold">
                                                {data.executionsToday}
                                            </p>
                                            <p className="text-xs text-muted-foreground">
                                                executions{data.failedExecutionsToday > 0 ? `, ${data.failedExecutionsToday} failed` : ""}
                                            </p>
                                        </div>
                                    </div>
                                </CardContent>
                            </Card>
                        </div>

                        {/* Pending Validations Alert */}
                        {data.pendingValidations > 0 && (
                            <Card className="border-blue-500/30 bg-blue-500/5">
                                <CardContent className="py-4">
                                    <div className="flex items-center gap-3">
                                        <Clock className="h-5 w-5 text-blue-500" />
                                        <span className="text-sm">
                                            <strong>{data.pendingValidations}</strong> article
                                            {data.pendingValidations !== 1 ? "s" : ""} pending
                                            validation
                                        </span>
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Content Statistics Section */}
                        <DashboardCharts />

                        {/* AI Usage Section */}
                        <AIUsageSection />
                    </motion.div>
                )}
            </AnimatePresence>

            {/* Loading State */}
            {isLoading && !data && (
                <div className="space-y-8">
                    <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
                        {[...Array(4)].map((_, i) => (
                            <div
                                key={i}
                                className="h-24 bg-muted/30 rounded-lg animate-pulse"
                                style={{ animationDelay: `${i * 0.05}s` }}
                            />
                        ))}
                    </div>
                    <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
                        {[...Array(4)].map((_, i) => (
                            <div
                                key={i}
                                className="h-24 bg-muted/30 rounded-lg animate-pulse"
                                style={{ animationDelay: `${i * 0.05}s` }}
                            />
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
