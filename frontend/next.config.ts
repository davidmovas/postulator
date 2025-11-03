import type { NextConfig } from "next";

const nextConfig: NextConfig = {
    /* config options here */
    // Added back static export for Wails build
    distDir: 'out',
    trailingSlash: true,
    allowedDevOrigins: ["wails.localhost"]
};

export default nextConfig;
