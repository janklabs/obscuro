/** @type {import("prettier").Config} */
const config = {
  semi: false,
  plugins: [
    "@ianvs/prettier-plugin-sort-imports",
    "prettier-plugin-tailwindcss",
    "prettier-plugin-packagejson",
  ],
  importOrder: [
    "^react$",
    "^next(/.*)?$",
    "<THIRD_PARTY_MODULES>",
    "",
    "^@/(.*)$",
    "^[./]",
  ],
}

export default config
