# website/src/app/

Next.js 16 App Router. One marketing landing page (`page.tsx`) + one docs catch-all (`docs/[[...slug]]/page.tsx`) + one search route (`api/search/route.ts`).

## Layout

```
app/
├── layout.tsx                 # Root: loads 4 local font families, sets <html>, wraps fumadocs RootProvider
├── page.tsx                   # Marketing homepage (hero + waves + features + how-it-works + footer)
├── globals.css                # @import "tailwindcss" — sole stylesheet
├── favicon.ico
├── fonts/                     # 7 .woff2 files (Satoshi, ClashDisplay, ClashGrotesk, IosevkaSS02 ×4 weights)
├── _components/               # PAGE-LOCAL components (underscore = excluded from routing)
│   ├── nav.tsx, footer.tsx, code-block.tsx, copy-button.tsx, feature-grid.tsx,
│   ├── glow-card.tsx, icons.tsx, section-heading.tsx, terminal-animation.tsx
│   ├── how-it-works/           # multi-file feature; index + sub-parts
│   └── waves/                  # canvas wave background animation
├── docs/
│   ├── layout.tsx             # fumadocs <DocsLayout> with sidebar from src/lib/source.ts
│   └── [[...slug]]/page.tsx   # Optional-catch-all → resolves MDX via fumadocs `source.getPage(slug)`
└── api/search/route.ts        # fumadocs flexsearch handler — exports GET
```

## Conventions

- `_components/` (underscore prefix) is the **only** sanctioned home for non-routing components used by pages here. Shared/cross-route UI goes to `src/components/ui/` (shadcn) instead.
- Fonts are loaded via `next/font/local` in `layout.tsx` and exposed as CSS variables — reference them from Tailwind classes (`font-sans`, `font-display`, `font-mono`), never via raw `font-family` declarations.
- The docs route uses Next 16's **double-bracket optional catch-all** `[[...slug]]` — `app/docs` and `app/docs/installation` both resolve here. Do not add `app/docs/page.tsx`; it will conflict.
- `api/search/route.ts` exports the fumadocs handler unchanged — if you customize search, extend the loader in `src/lib/source.ts` rather than editing the route.

## Anti-patterns

- **Do not** add `page.tsx` next to `[[...slug]]/page.tsx` — Next will throw an "intercepted route" build error.
- **Do not** mix `"use client"` into `layout.tsx` — root layout must stay a Server Component (font loading depends on it).
- **Do not** import font files directly in components — go through `layout.tsx` so they're hoisted into `<head>`.
- **Do not** drop a folder without an underscore prefix here unless you intend it to become a route (App Router treats every dir as a route segment).
