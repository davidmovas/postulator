"use client";

import { motion } from "framer-motion";

export function AnimatedHeader() {
    return (
        <motion.div
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="text-center space-y-2 mb-8"
        >
            <h1 className="text-4xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground/90 to-foreground/80 bg-clip-text text-transparent">
                Dashboard
            </h1>
            <p className="text-muted-foreground text-lg">
                Real-time overview of your content ecosystem
            </p>
        </motion.div>
    );
}