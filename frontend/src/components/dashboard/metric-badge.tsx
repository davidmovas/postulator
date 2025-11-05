"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface MetricBadgeProps {
    value: number;
    label: string;
    color: "green" | "red" | "blue" | "orange";
    delay?: number;
}

const colorStyles = {
    green: "bg-green-500/20 text-green-700 border-green-300",
    red: "bg-red-500/20 text-red-700 border-red-300",
    blue: "bg-blue-500/20 text-blue-700 border-blue-300",
    orange: "bg-orange-500/20 text-orange-700 border-orange-300"
};

export function MetricBadge({ value, label, color, delay = 0 }: MetricBadgeProps) {
    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.5, delay }}
            className={cn(
                "px-4 py-3 rounded-2xl border backdrop-blur-sm",
                colorStyles[color]
            )}
        >
            <div className="text-2xl font-bold">{value}</div>
            <div className="text-sm opacity-80">{label}</div>
        </motion.div>
    );
}