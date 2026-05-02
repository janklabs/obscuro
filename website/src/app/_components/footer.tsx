"use client"

import Link from "next/link"
import {
  BookOpen,
  FileText,
  Lock,
  Settings,
  Ship,
  Tag,
  Terminal,
} from "lucide-react"

import { GitHubIcon } from "./icons"

const footerLinks = [
  {
    title: "Documentation",
    links: [
      {
        label: "Getting Started",
        href: "/docs",
        icon: BookOpen,
        external: false,
      },
      {
        label: "Installation",
        href: "/docs/installation",
        icon: Terminal,
        external: false,
      },
      {
        label: "Configuration",
        href: "/docs/configuration",
        icon: Settings,
        external: false,
      },
      {
        label: "Helm Integration",
        href: "/docs/guides/helm-integration",
        icon: Ship,
        external: false,
      },
      {
        label: "Security",
        href: "/docs/security",
        icon: Lock,
        external: false,
      },
      {
        label: "Storage",
        href: "/docs/storage",
        icon: FileText,
        external: false,
      },
    ],
  },
  {
    title: "GitHub",
    links: [
      {
        label: "Repository",
        href: "https://github.com/janklabs/obscuro",
        icon: GitHubIcon,
        external: true,
      },
      {
        label: "Releases",
        href: "https://github.com/janklabs/obscuro/releases",
        icon: Tag,
        external: true,
      },
      {
        label: "Issues",
        href: "https://github.com/janklabs/obscuro/issues",
        icon: FileText,
        external: true,
      },
    ],
  },
]

function FooterLink({
  href,
  label,
  icon: Icon,
  external,
}: {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  external: boolean
}) {
  const externalProps = external
    ? { target: "_blank", rel: "noopener noreferrer" }
    : {}
  return (
    <Link
      href={href}
      className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-xs transition-colors duration-200"
      {...(externalProps as Record<string, string>)}
    >
      <Icon className="size-3.5" />
      {label}
    </Link>
  )
}

function FooterLinkHeader({ title }: { title: string }) {
  return (
    <h3 className="text-muted-foreground mb-4 text-xs font-bold tracking-[0.2em] uppercase">
      {title}
    </h3>
  )
}

export function Footer() {
  return (
    <footer className="border-border relative z-10 border-t">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <div className="grid grid-cols-2 gap-10">
          {/* Brand */}
          <div className="flex flex-col gap-8">
            <div>
              <FooterLinkHeader title="GitHub" />
              <div className="flex flex-col gap-2">
                {footerLinks[1].links.map((link) => (
                  <FooterLink key={link.href} {...link} />
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-3">
              <div className="flex items-center gap-2.5">
                <span className="bg-terminal inline-block size-2.5 rounded-full shadow-[0_0_8px_rgba(191,90,242,0.4)]" />
                <span className="text-foreground text-sm font-bold tracking-widest uppercase">
                  Obscuro
                </span>
              </div>
              <p className="text-muted-foreground max-w-xs text-xs/relaxed">
                Encrypt secrets. Commit safely. Deploy anywhere.
              </p>
              <div className="text-muted-foreground text-xs">
                <span>&copy; {new Date().getFullYear()} Jank Labs</span>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-8">
            <div>
              <FooterLinkHeader title="Documentation" />
              <div className="flex flex-col gap-2">
                {footerLinks[0].links.map((link) => (
                  <FooterLink key={link.href} {...link} />
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </footer>
  )
}
