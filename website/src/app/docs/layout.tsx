import type { ReactNode } from "react"
import { DocsLayout } from "fumadocs-ui/layouts/docs"

import { source } from "@/lib/source"

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={source.pageTree}
      nav={{
        title: (
          <span className="flex items-center gap-2">
            <span className="bg-terminal inline-block size-2 rounded-full shadow-[0_0_6px_rgba(191,90,242,0.4)]" />
            <span className="text-xs font-bold tracking-widest uppercase">
              Obscuro
            </span>
          </span>
        ),
        url: "/",
      }}
    >
      {children}
    </DocsLayout>
  )
}
