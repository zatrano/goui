@echo off
REM Optional: build Tailwind utilities from input.css
REM Requires: npx (or standalone tailwindcss binary)
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
echo Built output.css — point index.html at it if you want Tailwind-generated utilities.
