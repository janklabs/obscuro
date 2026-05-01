import { ArrowDown, ArrowRight } from "lucide-react"

import { cn } from "@/lib/utils"

export function WindowChrome({
  filename,
  children,
  accent = false,
  className,
}: {
  filename: string
  children: React.ReactNode
  accent?: boolean
  className?: string
}) {
  return (
    <div
      className={cn(
        "border-border bg-card overflow-hidden rounded-lg border",
        className,
      )}
    >
      {/* Title bar */}
      <div className="border-border flex items-center gap-2 border-b px-3 py-2">
        <span className="size-2 rounded-full bg-[#ff5f57]" />
        <span className="size-2 rounded-full bg-[#febc2e]" />
        <span className="size-2 rounded-full bg-[#28c840]" />
        <span className="text-muted-foreground ml-1 font-mono text-[11px]">
          {filename}
        </span>
      </div>
      {/* Content */}
      <div
        className={cn("p-4 font-mono text-xs/relaxed", accent && "bg-card/80")}
      >
        {children}
      </div>
    </div>
  )
}

export function TerminalLine({ command }: { command: string }) {
  return (
    <div className="flex items-start gap-2 font-mono text-xs">
      <span className="text-terminal select-none">$</span>
      <code className="text-muted-foreground">{command}</code>
    </div>
  )
}

export function Connector({
  direction = "right",
}: {
  direction?: "right" | "down"
}) {
  if (direction === "down") {
    return (
      <div className="flex justify-center py-4 md:hidden">
        <ArrowDown className="text-terminal/30 size-4" />
      </div>
    )
  }
  return (
    <div className="hidden items-center justify-center md:flex">
      <ArrowRight className="text-terminal/30 size-5" />
    </div>
  )
}

export function StepLabel({
  number,
  title,
}: {
  number: string
  title: string
}) {
  return (
    <div className="mb-3 flex items-center gap-3">
      <span className="text-terminal font-mono text-xs font-bold">
        {number}
      </span>
      <h3
        className="text-foreground text-sm font-bold tracking-tight"
        style={{ fontFamily: "var(--font-display)" }}
      >
        {title}
      </h3>
    </div>
  )
}

export function YamlLine({
  indent = 0,
  content,
  highlight = false,
}: {
  indent?: number
  content: React.ReactNode
  highlight?: boolean
}) {
  return (
    <div className={cn(highlight && "bg-terminal/8 -mx-2 rounded-sm px-2")}>
      <span className="text-muted-foreground whitespace-pre">
        {"  ".repeat(indent)}
      </span>
      {content}
    </div>
  )
}

export function YamlKey({ k }: { k: string }) {
  return <span className="text-warm">{k}</span>
}

export function YamlString({ v }: { v: string }) {
  return <span className="text-[#a5d6a7]">{v}</span>
}

export function YamlPlaceholder({ v }: { v: string }) {
  return <span className="text-terminal font-bold">{v}</span>
}
