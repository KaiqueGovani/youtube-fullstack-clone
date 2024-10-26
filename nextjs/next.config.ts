import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  images: {
    domains: ["host.docker.internal", "localhost", "nginx"],
  },
  experimental: {
    after: true,
  },
};

export default nextConfig;
