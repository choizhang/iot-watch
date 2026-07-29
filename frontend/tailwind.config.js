/** @type {import('tailwindcss').Config} */

export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,vue}"],
  theme: {
    container: {
      center: true,
    },
    extend: {
      animation: {
        'bounce-slow': 'bounce 2s infinite',
      },
    }
  },
  plugins: [],
};
