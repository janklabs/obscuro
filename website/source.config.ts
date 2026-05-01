import { rehypeCode } from "fumadocs-core/mdx-plugins/rehype-code"
import { defineConfig, defineDocs } from "fumadocs-mdx/config"

export const docs = defineDocs({
  dir: "content/docs",
})

export default defineConfig({
  mdxOptions: {
    rehypePlugins: [
      [
        rehypeCode,
        {
          themes: {
            light: "vitesse-dark",
            dark: "vitesse-dark",
          },
        },
      ],
    ],
  },
})
