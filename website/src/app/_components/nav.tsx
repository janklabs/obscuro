"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { BookOpen, Menu, X } from "lucide-react"

import { cn } from "@/lib/utils"
import { GitHubIcon } from "./icons"

export function Nav() {
  const [scrolled, setScrolled] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 10)
    onScroll()
    window.addEventListener("scroll", onScroll, { passive: true })
    return () => window.removeEventListener("scroll", onScroll)
  }, [])

  return (
    <header
      className={cn(
        "bg-background/80 fixed inset-x-0 top-0 z-50 border-b backdrop-blur-md transition-colors duration-300",
        scrolled ? "border-border" : "border-transparent",
      )}
    >
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
        <Link href="/" className="group flex items-center gap-2.5">
          <span className="bg-terminal inline-block size-2.5 rounded-full shadow-[0_0_8px_rgba(191,90,242,0.4)] transition-shadow duration-300 group-hover:shadow-[0_0_14px_rgba(191,90,242,0.6)]" />
          <span className="text-foreground text-sm font-bold tracking-widest uppercase">
            Obscuro
          </span>
        </Link>

        {/* Desktop nav */}
        <nav className="items-center gap-6 sm:flex">
          <Link
            href="/docs"
            className="text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs tracking-wide transition-colors duration-200"
          >
            <BookOpen className="size-3.5" />
            Docs
          </Link>
          <Link
            href="https://github.com/janklabs/obscuro"
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs tracking-wide transition-colors duration-200"
          >
            <GitHubIcon className="size-3.5" />
            GitHub
          </Link>
        </nav>

        {/* Mobile hamburger */}
        <button
          className="text-muted-foreground hover:text-foreground inline-flex cursor-pointer items-center justify-center rounded-sm p-1.5 transition-colors sm:hidden"
          onClick={() => setMobileOpen(!mobileOpen)}
          aria-label={mobileOpen ? "Close menu" : "Open menu"}
        >
          {mobileOpen ? <X className="size-5" /> : <Menu className="size-5" />}
        </button>
      </div>

      {/* Mobile menu */}
      {mobileOpen && (
        <nav className="border-border bg-background/95 border-t px-6 py-4 backdrop-blur-md sm:hidden">
          <div className="flex flex-col gap-4">
            <Link
              href="/docs"
              onClick={() => setMobileOpen(false)}
              className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-sm transition-colors"
            >
              <BookOpen className="size-4" />
              Docs
            </Link>
            <Link
              href="https://github.com/janklabs/obscuro"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setMobileOpen(false)}
              className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-sm transition-colors"
            >
              <GitHubIcon className="size-4" />
              GitHub
            </Link>
          </div>
        </nav>
      )}
    </header>
  )
}
