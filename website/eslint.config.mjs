import nextVitals from "eslint-config-next/core-web-vitals"
import nextTs from "eslint-config-next/typescript"
import betterTailwindcss from "eslint-plugin-better-tailwindcss"
import reactCompiler from "eslint-plugin-react-compiler"
import { defineConfig, globalIgnores } from "eslint/config"

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  // Tailwind CSS correctness + stylistic rules
  {
    ...betterTailwindcss.configs["recommended"],
    settings: {
      "better-tailwindcss": {
        entryPoint: "src/app/globals.css",
      },
    },
  },

  // Allowlist custom CSS classes
  {
    rules: {
      "better-tailwindcss/no-unknown-classes": [
        "error",
        {
          ignore: [
            "dark",
            "animate-fade-up",
            "cursor-blink",
            "text-glow",
            "noise-bg",
            "scanlines",
            "waves",
            "waves-canvas",
          ],
        },
      ],
      // Disable line-wrapping — its auto-fix creates template literals which
      // conflict with the no-restricted-syntax ban on template literals in className.
      "better-tailwindcss/enforce-consistent-line-wrapping": "off",
      // Disable class ordering — prettier-plugin-tailwindcss handles this and
      // conflicts with the eslint plugin's ordering.
      "better-tailwindcss/enforce-consistent-class-order": "off",
    },
  },

  // React Compiler — catches unnecessary useEffect and other patterns
  {
    plugins: { "react-compiler": reactCompiler },
    rules: {
      "react-compiler/react-compiler": "error",
    },
  },

  // Ban template literals in className — use cn() instead
  {
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "JSXAttribute[name.name='className'] TemplateLiteral",
          message:
            "Don't use template literals in className. Use cn() from @/lib/utils instead.",
        },
      ],
    },
  },

  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    ".open-next/**",
    ".wrangler/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    ".source/**",
  ]),
])

export default eslintConfig
