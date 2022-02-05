const colors = require('tailwindcss/colors')

module.exports = {
    content: [
        './src/**/*.{tsx,ts,html,css}',
    ],
    // purge: false,
    darkMode: 'class', // or 'media' or 'class'
    theme: {
        extend: {
            colors: {
                gray: colors.neutral,
            },
        },
    },
    variants: {
        extend: {},
    },
    plugins: [
        require('@tailwindcss/forms'),
    ],
}
