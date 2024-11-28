/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./public/modules/**/*.js", "./**/*.html", "./**/*.go", "./**/*.templ"],
  safelist: [],
  theme: {
    extend: {
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
      animation: {
        fadeIn: 'fadeIn 0.3s ease-in-out forwards',
      },
    },
  },
};
