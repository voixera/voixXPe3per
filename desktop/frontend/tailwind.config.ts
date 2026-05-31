import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        shell: {
          950: "#0f1013",
          900: "#14161a",
          850: "#191c21",
          800: "#1f2329",
          700: "#2a3038",
          600: "#3a424d"
        },
        signal: {
          green: "#4fd18b",
          amber: "#f3b454",
          red: "#ff6b6b",
          cyan: "#66d9ef"
        }
      },
      boxShadow: {
        insetPanel: "inset 0 1px 0 rgba(255,255,255,0.04)"
      },
      fontFamily: {
        sans: ["Inter", "Segoe UI", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "Consolas", "monospace"]
      }
    }
  },
  plugins: []
} satisfies Config;
