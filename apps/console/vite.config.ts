import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const registryTarget = process.env.ARD_CONSOLE_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/admin": registryTarget,
      "/agents": registryTarget,
      "/explore": registryTarget,
      "/health": registryTarget,
      "/metrics": registryTarget,
      "/search": registryTarget,
      "/.well-known": registryTarget
    }
  }
});
