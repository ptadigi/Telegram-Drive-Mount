import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";
import react from "@vitejs/plugin-react";
import Icons from "unplugin-icons/vite";

export default defineConfig({
  plugins: [
    react(),
    // Compile Iconify (Phosphor) icons to inline SVG React components at build
    // time — no runtime network fetch, works offline / self-hosted.
    Icons({ compiler: "jsx", jsx: "react", autoInstall: false }),
  ],
  resolve: {
    alias: {
      // The whole app imports icons from "lucide-react"; redirect that to our
      // central Phosphor adapter so we keep ONE icon set without editing every
      // component (and without risking text-encoding damage).
      "lucide-react": fileURLToPath(new URL("./src/icons.tsx", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
});
