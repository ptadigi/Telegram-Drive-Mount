import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import Icons from "unplugin-icons/vite";

export default defineConfig({
  plugins: [
    react(),
    // Compile Iconify (Solar) icons to inline SVG React components at build
    // time — no runtime network fetch, works offline / self-hosted.
    Icons({ compiler: "jsx", jsx: "react", autoInstall: false }),
  ],
  server: {
    port: 5173,
    strictPort: true,
  },
});
