import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "#fcfcfb",
        ink: "#0b0b0b",
        primary: "#0b0b0b",
        muted: "#898781",
        line: "rgba(11,11,11,0.1)",
        clay: "#c6613f",
        fill: "#f2f1ee"
      },
      borderRadius: {
        cds: "8px",
        control: "7px"
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "\"Segoe UI\"", "Roboto", "Helvetica", "Arial", "sans-serif"],
        serif: ["Georgia", "\"Times New Roman\"", "serif"],
        voice: ["Georgia", "\"Times New Roman\"", "serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"]
      }
    }
  },
  plugins: []
} satisfies Config;
