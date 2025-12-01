import type { NextConfig } from "next";

const nextConfig: NextConfig = {
    output: 'export',
    distDir: 'out',
    trailingSlash: true,

    // Dev settings
    allowedDevOrigins: ["wails.localhost"],

    // Optimize images for static hosting
    images: {
        unoptimized: true,
    },

    // Skip type checking and linting during build (faster builds)
    typescript: {
        ignoreBuildErrors: true,
    },
    eslint: {
        ignoreDuringBuilds: true,
    },
};

export default nextConfig;
