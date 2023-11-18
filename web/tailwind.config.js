import { lerpColors } from "tailwind-lerp-colors";
import plugin from "tailwindcss/plugin";

const extendedColors = lerpColors();

module.exports = {
  content: [
    "./src/**/*.{tsx,ts,html,css}",
  ],
  safelist: [
    "col-span-1",
    "col-span-2",
    "col-span-3",
    "col-span-4",
    "col-span-5",
    "col-span-6",
    "col-span-7",
    "col-span-8",
    "col-span-9",
    "col-span-10",
    "col-span-11",
    "col-span-12",
  ],
  // purge: false,
  darkMode: "class", // or 'media' or 'class'
  theme: {
    extend: {
      colors: {
        ...extendedColors,
        gray: {
          ...extendedColors.zinc,
          815: "#232427"
        }
      },
      margin: { // for the checkmarks used for regex validation in Filters/Advanced
        "2.5": "0.625rem", // 10px, between mb-2 (8px) and mb-3 (12px)
      },
      textShadow: {
        DEFAULT: "0 2px 4px var(--tw-shadow-color)"
      },
      boxShadow: {
        table: "rgba(0, 0, 0, 0.1) 0px 4px 16px 0px"
      }
    },
  },
  variants: {
    extend: {},
  },
  plugins: [
    require("@tailwindcss/forms"),
    plugin(function ({ matchUtilities, theme }) {
      // Pipe --tw-shadow-color (i.e. shadow-cyan-500/50) to our new text-shadow
      // Credits: https://www.hyperui.dev/blog/text-shadow-with-tailwindcss
      // Use it like: text-shadow shadow-cyan-500/50
      matchUtilities(
        { "text-shadow": (value) => ({ textShadow: value }) },
        { values: theme("textShadow") }
      );
    }),
  ],
}
