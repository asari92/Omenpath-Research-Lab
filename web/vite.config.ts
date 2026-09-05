import react from "@vitejs/plugin-react";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const backendTarget =
  process.env.VITE_BACKEND_TARGET ?? "http://127.0.0.1:8080";
const worklogPath = fileURLToPath(
  new URL("../01_AI_WORKLOG_CURRENT.md", import.meta.url),
);

export default defineConfig({
  plugins: [
    {
      name: "single-root-worklog",
      enforce: "pre",
      resolveId(source) {
        return source === "../../../01_AI_WORKLOG_CURRENT.md?raw"
          ? "\0virtual:omenpath-worklog"
          : null;
      },
      load(id) {
        return id === "\0virtual:omenpath-worklog"
          ? `export default ${JSON.stringify(readFileSync(worklogPath, "utf8"))}`
          : null;
      },
    },
    react(),
  ],
  server: {
    proxy: {
      "/api": backendTarget,
      "/ws": { target: backendTarget, ws: true },
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.{ts,tsx}"],
    setupFiles: ["./src/test/setup.ts"],
    css: true,
  },
});
