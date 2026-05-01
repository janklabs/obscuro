import { cn } from "@/lib/utils"
import { CopyButton } from "./copy-button"

interface CodeBlockProps {
  code: string
  /** Show a $ prompt before the code. Default true */
  prompt?: boolean
  className?: string
}

export function CodeBlock({ code, prompt = true, className }: CodeBlockProps) {
  return (
    <div
      className={cn(
        "border-border bg-card flex max-w-full items-center gap-3 overflow-x-auto rounded-lg border px-4 py-3 font-mono text-xs",
        className,
      )}
    >
      {prompt && <span className="text-terminal shrink-0 select-none">$</span>}
      <code className="text-muted-foreground whitespace-nowrap">{code}</code>
      <CopyButton text={code} />
    </div>
  )
}
