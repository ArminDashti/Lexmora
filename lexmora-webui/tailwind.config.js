/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Vazirmatn', 'system-ui', 'sans-serif'],
      },
      colors: {
        surface: {
          DEFAULT: '#1e1e1e',
          raised: '#252526',
          border: '#3c3c3c',
        },
        accent: {
          DEFAULT: '#007acc',
          muted: '#005a9e',
        },
      },
    },
  },
  plugins: [],
}
