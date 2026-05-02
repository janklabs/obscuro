"use client"

import { useEffect, useRef, useState } from "react"

interface TerminalLine {
  type: "command" | "output" | "success" | "password"
  text: string
}

const SCRIPT: TerminalLine[] = [
  { type: "command", text: "obscuro init" },
  { type: "password", text: "Master password: ••••••••••" },
  { type: "password", text: "Confirm password: ••••••••••" },
  { type: "success", text: "✓ Vault initialized at .obscuro/" },
  { type: "command", text: "obscuro set API_KEY --value sk-live-abc123" },
  { type: "success", text: '✓ Secret "API_KEY" stored' },
  {
    type: "command",
    text: "helm install myapp ./chart --post-renderer obscuro",
  },
  { type: "success", text: "✓ Secrets injected into manifests" },
]

// Delays between lines appearing (ms)
const PAUSE_AFTER_COMMAND = 400
const PAUSE_AFTER_OUTPUT = 150
const PAUSE_BEFORE_COMMAND = 800
const TYPE_SPEED = 28

export function TerminalAnimation() {
  const [lines, setLines] = useState<
    { idx: number; typed: string; done: boolean }[]
  >([])
  const [finished, setFinished] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const abortController = new AbortController()
    const cancelled = () => abortController.signal.aborted

    async function run() {
      for (let i = 0; i < SCRIPT.length; i++) {
        if (cancelled()) return
        const line = SCRIPT[i]

        if (line.type === "command") {
          if (i > 0) await sleep(PAUSE_BEFORE_COMMAND)
          if (cancelled()) return

          setLines((prev) => [...prev, { idx: i, typed: "", done: false }])

          for (let c = 1; c <= line.text.length; c++) {
            if (cancelled()) return
            await sleep(TYPE_SPEED)
            if (cancelled()) return
            const chars = c
            setLines((prev) =>
              prev.map((l) =>
                l.idx === i ? { ...l, typed: line.text.slice(0, chars) } : l,
              ),
            )
          }

          setLines((prev) =>
            prev.map((l) => (l.idx === i ? { ...l, done: true } : l)),
          )

          await sleep(PAUSE_AFTER_COMMAND)
        } else {
          await sleep(PAUSE_AFTER_OUTPUT)
          if (cancelled()) return
          setLines((prev) => [
            ...prev,
            { idx: i, typed: line.text, done: true },
          ])
        }
      }

      if (!cancelled()) setFinished(true)
    }

    run()
    return () => {
      abortController.abort()
    }
  }, [])

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [lines.length])

  return (
    <div className="relative overflow-hidden rounded-lg border border-white/10 bg-black/60 shadow-[0_0_80px_rgba(191,90,242,0.08)] backdrop-blur-md">
      {/* SVG filter definition (hidden) */}
      <svg className="absolute size-0" aria-hidden="true">
        <filter id="terminal-grain">
          <feTurbulence
            type="fractalNoise"
            baseFrequency="0.85"
            numOctaves="4"
            stitchTiles="stitch"
          />
        </filter>
      </svg>
      {/* Grain overlay */}
      <div
        className="pointer-events-none absolute inset-0 z-10"
        style={{
          filter: "url(#terminal-grain)",
          opacity: 0.35,
          mixBlendMode: "overlay",
        }}
      />

      {/* Title bar */}
      <div className="relative z-20 flex items-center gap-2 border-b border-white/10 bg-black/40 px-4 py-3">
        <div className="size-3 rounded-full bg-[#ff5f57]" />
        <div className="size-3 rounded-full bg-[#febc2e]" />
        <div className="size-3 rounded-full bg-[#28c840]" />
        <span className="ml-3 font-mono text-xs text-[#555]">~/.obscuro</span>
      </div>

      {/* Terminal body */}
      <div
        ref={containerRef}
        className="relative z-20 h-[300px] overflow-hidden p-5 font-mono text-sm/relaxed"
      >
        {lines.map((entry) => {
          const line = SCRIPT[entry.idx]
          return (
            <div key={entry.idx} className="flex items-start">
              {line.type === "command" ? (
                <>
                  <span className="text-terminal mr-2 shrink-0 select-none">
                    $
                  </span>
                  <span className="text-[#e0e0e0]">
                    {entry.typed}
                    {!entry.done && (
                      <span className="cursor-blink bg-terminal ml-px inline-block h-[14px] w-[7px] align-middle" />
                    )}
                  </span>
                </>
              ) : line.type === "success" ? (
                <span className="text-terminal">{entry.typed}</span>
              ) : line.type === "password" ? (
                <span className="text-[#888]">{entry.typed}</span>
              ) : (
                <span className="pl-4 text-[#999]">{entry.typed}</span>
              )}
            </div>
          )
        })}
        {/* Permanent blinking cursor after animation completes */}
        {finished && (
          <div className="flex items-start">
            <span className="text-terminal mr-2 shrink-0">$</span>
            <span className="cursor-blink bg-terminal inline-block h-[14px] w-[7px]" />
          </div>
        )}
      </div>
    </div>
  )
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
