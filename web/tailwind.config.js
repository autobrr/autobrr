const colors = require('tailwindcss/colors')

module.exports = {
    // mode: 'jit',
    purge: {
        content: [
            './src/**/*.{tsx,ts,html,css}',
        ],
        safelist: [
            'col-span-1',
            'col-span-2',
            'col-span-3',
            'col-span-4',
            'col-span-5',
            'col-span-6',
            'col-span-7',
            'col-span-8',
            'col-span-9',
            'col-span-10',
            'col-span-11',
            'col-span-12',
        ],
    },
    darkMode: 'media', // or 'media' or 'class'
    theme: {
        extend: {
            colors: {
                gray: colors.gray,
                teal: colors.teal,
            }
        },
    },
    variants: {
        extend: {},
    },
    plugins: [
        require('@tailwindcss/forms'),
    ],
}
