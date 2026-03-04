import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
    build: {
        outDir: "dist",
        emptyOutDir: true,
    },
    server: {
        proxy: {
            "/api": {
                target: `http://localhost:${process.env.QWELLI_PORT ?? 8080}`,
                changeOrigin: true,
            },
        },
    },
});
