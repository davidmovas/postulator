import type { NextConfig } from "next";

const nextConfig: NextConfig = {
    /* config options here */
    // Removed static export to support dynamic routes like /topics/[siteId]
    allowedDevOrigins: ["wails.localhost"]
};

export default nextConfig;
