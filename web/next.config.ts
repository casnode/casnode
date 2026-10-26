import type { NextConfig } from "next";

// `next build` writes a Next.js server to .next-build/standalone, which scripts/prepare-standalone.mjs moves to a new
// web/build/server-<time> on every build: the Go backend in the repository root starts the newest one and sends the
// pages to it, and a build never has to delete the server that is running.
const backendUrl = process.env.BACKEND_URL || "http://127.0.0.1:16900";

const nextConfig: NextConfig = {
  output: "standalone",
  distDir: ".next-build",
  images: { unoptimized: true },
  // the backend compresses the responses
  compress: false,
  poweredByHeader: false,
  // with `yarn dev` the backend runs on its own port; in production it sits in front of Next and answers these itself
  async rewrites() {
    return {
      beforeFiles: [
        { source: "/api/:path*", destination: `${backendUrl}/api/:path*` },
        { source: "/uploads/:path*", destination: `${backendUrl}/uploads/:path*` },
        { source: "/robots.txt", destination: `${backendUrl}/robots.txt` },
        { source: "/sitemap.xml", destination: `${backendUrl}/sitemap.xml` },
        { source: "/:file(sitemap_\d+\.xml)", destination: `${backendUrl}/:file` },
        { source: "/:path(.*\.rss)", destination: `${backendUrl}/:path` },
      ],
      afterFiles: [],
      fallback: [],
    };
  },
};

export default nextConfig;
