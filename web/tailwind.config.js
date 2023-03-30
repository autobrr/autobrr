const colors = require("tailwindcss/colors")

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
        gray: colors.zinc,
      },
      margin: { // for the checkmarks used for regex validation in Filters/Advanced
        "2.5": "0.625rem", // 10px, between mb-2 (8px) and mb-3 (12px)
      },
    },
  },
  variants: {
    extend: {},
  },
  plugins: [
    require("@tailwindcss/forms"),
  ],
}
