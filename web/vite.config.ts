import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "127.0.0.1",
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:4780",
        changeOrigin: true,
        configure: (p) => p.on("proxyReq", (req) => req.removeHeader("origin")),
      },
      "/covers": { target: "http://127.0.0.1:4780", changeOrigin: true },
    },
  },
  test: { environment: "jsdom", setupFiles: ["./src/test-setup.ts"] },
});
