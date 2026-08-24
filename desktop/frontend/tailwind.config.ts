import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#070807",
          900: "#0a0b09",
          850: "#0e100d",
          800: "#12140f",
          700: "#191c15",
          600: "#23271e",
          500: "#323827"
        },
        line: {
          dim: "#1c2018",
          mid: "#2c3325",
          hi: "#46503a"
        },
        acid: {
          DEFAULT: "#c6f542",
          soft: "#dcff7a"
        },
        bone: "#e6ecdd",
        dim: "#7d8570",
        amber: "#ffb224",
        alarm: "#ff5d45"
      },
      fontFamily: {
        display: ["Bahnschrift", "Arial Narrow", "Segoe UI", "sans-serif"],
        mono: ["JetBrains Mono", "Consolas", "ui-monospace", "monospace"]
      }
    }
  },
  plugins: []
} satisfies Config;
