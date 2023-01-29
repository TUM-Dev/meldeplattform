/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./templates/*.gohtml", "src/*.ts"],
  theme: {
    extend: {
      container: {
        center: true,
        padding: {
          DEFAULT: "1rem",
          sm: "2rem",
          lg: "6rem",
          xl: "8rem",
          "2xl": "10rem",
        },
      },
    },
  },
  plugins: [],
}
