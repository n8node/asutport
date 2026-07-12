import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        bg: "var(--bg)",
        panel: "var(--panel)",
        panel2: "var(--panel2)",
        line: "var(--line)",
        line2: "var(--line2)",
        text: "var(--text)",
        mut: "var(--mut)",
        dim: "var(--dim)",
        acc: "var(--acc)",
        lampGreen: "var(--green)",
        lampAmber: "var(--amber)",
        lampRed: "var(--red)",
      },
      fontFamily: {
        logo: ["var(--font-unbounded)", "sans-serif"],
        sans: ["var(--font-onest)", "sans-serif"],
        mono: ["var(--font-jetbrains)", "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;
