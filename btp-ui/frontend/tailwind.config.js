/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./lib/**/*.{js,ts,jsx,tsx,mdx}"
  ],
  theme: {
    extend: {
      colors: {
        background: "#f5f7fb",
        card: "#ffffff",
        primary: "#2f6bff",
        success: "#17b26a",
        warning: "#f7b801",
        danger: "#ef4444",
        muted: "#64748b"
      },
      boxShadow: {
        soft: "0 10px 30px rgba(15, 23, 42, 0.06)",
        glow: "0 0 0 1px rgba(47, 107, 255, 0.18), 0 18px 40px rgba(47, 107, 255, 0.18)"
      },
      borderRadius: {
        xl2: "1.25rem"
      }
    }
  },
  plugins: []
};
