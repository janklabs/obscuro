# website/content/docs/

MDX content for the docs site. Loaded by `fumadocs-mdx` via `source.config.ts`; surfaced through `src/lib/source.ts`. **Edit MDX here, never the generated `.source/` dir.**

## Layout

```
docs/
├── meta.json                # Top-level sidebar order (uses "---Label---" separators)
├── index.mdx                # / docs root — "Getting Started"
├── installation.mdx
├── configuration.mdx
├── security.mdx
├── storage.mdx
├── commands/                # One MDX per CLI verb (init, set, get, list, remove, edit, inject, auth, upgrade, version)
│   └── meta.json            # Sidebar order for this section
└── guides/                  # docker-compose, helm-integration, kubernetes
    └── meta.json
```

## meta.json

Drives sidebar grouping. The string `"---Label---"` (triple-dash bookended) renders as a non-clickable section heading. `"commands"` / `"guides"` entries auto-expand based on each subdir's own `meta.json`.

## Frontmatter (required on every .mdx)

```mdx
---
title: <H1 used in sidebar + <title>>
description: <SEO meta + page intro>
---
```

Missing either field breaks the fumadocs page renderer at build time.

## Conventions

- **Code fences** are highlighted at build time by `rehype-code` (theme `vitesse-dark`). Use language tags (` ```sh `, ` ```yaml `, ` ```go `) — bare `` ` `` won't highlight.
- Use `__PLACEHOLDER__` (double underscore) when illustrating `obscuro inject` — that is the literal substitution syntax the CLI looks for. Never use `${VAR}` or `{{var}}` in those examples; readers copy-paste.
- Inline `<Callout>` and other fumadocs MDX components are available **without import** (registered globally in `mdx-components.tsx`).
- Wikilink-style `[[page]]` is **not** supported — use standard markdown links: `[Init](/docs/commands/init)`.

## Anti-patterns

- **Do not** put real secrets in code samples (the `docker-compose.mdx` and `kubernetes.mdx` guides explicitly call this out — keep example values placeholder-shaped).
- **Do not** add an MDX file without registering it in the closest `meta.json` — it will route but be invisible in the sidebar.
- **Do not** rename a command page without updating the corresponding `cmd/<verb>.go` deep-link in the CLI's help text.
- **Do not** create a directory without a `meta.json` — fumadocs falls back to alphabetical, which has bitten us on the commands list.
