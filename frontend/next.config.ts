import type { NextConfig } from "next";

const noStoreHeaders = [
  { key: "Cache-Control", value: "no-store, no-cache, must-revalidate, max-age=0" },
  { key: "Pragma", value: "no-cache" },
  { key: "Expires", value: "0" },
];

const nextConfig: NextConfig = {
  output: "standalone",
  generateBuildId: async () => process.env.BUILD_ID || "development",
  async headers() {
    return [
      { source: "/dashboard", headers: noStoreHeaders },
      { source: "/dashboard/:path*", headers: noStoreHeaders },
      { source: "/vendor", headers: noStoreHeaders },
      { source: "/vendor/:path*", headers: noStoreHeaders },
      { source: "/admin", headers: noStoreHeaders },
      { source: "/admin/:path*", headers: noStoreHeaders },
    ];
  },
};

export default nextConfig;
