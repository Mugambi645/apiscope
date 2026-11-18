/** @type {import('next').NextConfig} */
const gateway = process.env.GATEWAY_URL || "http://localhost:8080";

export default {
  output: "standalone",
  reactStrictMode: true,
  poweredByHeader: false,
  // The browser talks to /api/*; Next forwards to the Go gateway, so the
  // gateway needs no public CORS and the web origin stays the only origin.
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${gateway}/api/:path*` }];
  },
};
