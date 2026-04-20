/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "#f5f1e8",
        ink: "#17221c",
        coral: "#db6c3f",
        sea: "#0c8f6d",
        sand: "#e7dbc2",
      },
      fontFamily: {
        display: ["Avenir Next", "Trebuchet MS", "sans-serif"],
        body: ["IBM Plex Sans", "Segoe UI", "sans-serif"],
      },
      boxShadow: {
        float: "0 24px 60px rgba(23, 34, 28, 0.12)",
      },
      backgroundImage: {
        grain:
          "radial-gradient(circle at top left, rgba(219,108,63,0.20), transparent 22rem), radial-gradient(circle at top right, rgba(12,143,109,0.18), transparent 26rem)",
      },
      animation: {
        "fade-up": "fade-up 700ms ease-out both",
      },
      keyframes: {
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(16px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
    },
  },
  plugins: [],
};

