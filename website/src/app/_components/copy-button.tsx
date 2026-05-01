"use client"

import { useState } from "react"
import { Check, Copy } from "lucide-react"

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

interface CopyButtonProps {
  text: string
  className?: string
}

export function CopyButton({ text, className }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Tooltip>
      <TooltipTrigger
        onClick={handleCopy}
        className={cn(
          "border-border bg-card hover:border-terminal inline-flex shrink-0 cursor-pointer items-center rounded-sm border p-1.5 transition-all duration-300 hover:shadow-[0_0_12px_rgba(191,90,242,0.15)]",
          className,
        )}
        aria-label="Copy to clipboard"
      >
        {copied ? (
          <Check className="text-terminal size-3.5" />
        ) : (
          <Copy className="text-muted-foreground size-3.5" />
        )}
      </TooltipTrigger>
      <TooltipContent>{copied ? "Copied!" : "Copy"}</TooltipContent>
    </Tooltip>
  )
}
