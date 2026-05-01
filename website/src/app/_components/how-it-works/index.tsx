"use client"

import { ArrowRight, Play } from "lucide-react"

import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll"
import { cn } from "@/lib/utils"
import {
  Connector,
  StepLabel,
  TerminalLine,
  WindowChrome,
  YamlKey,
  YamlLine,
  YamlPlaceholder,
  YamlString,
} from "./parts"

export function HowItWorks() {
  const { ref, isVisible } = useAnimateOnScroll()

  return (
    <div ref={ref} className="grid grid-cols-3 grid-rows-1 gap-4">
      {/* ─── PANEL 1: Store a secret ─── */}
      <div
        className={cn(isVisible ? "animate-fade-up" : "opacity-0")}
        style={{ animationDelay: "0s" }}
      >
        <StepLabel number="01" title="Store a secret" />
        <div className="flex flex-col gap-4">
          <div>Encrypt and store a secret</div>
          {/* Command */}
          <WindowChrome filename="terminal">
            <TerminalLine command='obscuro set API_KEY --value "sk-live-abc123"' />
          </WindowChrome>

          <div>Here&apos;s what actually gets committed</div>
          {/* Resulting secrets file */}
          <WindowChrome filename=".obscuro/secrets.json">
            <div className="text-muted-foreground">
              <div>{"{"}</div>
              <div className="ml-4">
                <YamlKey k='"API_KEY"' />
                <span className="text-muted-foreground">: </span>
                <YamlString v='"AES256:v1:a3F7x...9kWm="' />
              </div>
              <div>{"}"}</div>
            </div>
          </WindowChrome>
        </div>
      </div>

      <Connector direction="down" />

      {/* ─── PANEL 2: Your compose file ─── */}
      <div
        className={cn(isVisible ? "animate-fade-up" : "opacity-0")}
        style={{ animationDelay: "0.15s" }}
      >
        <StepLabel number="02" title="Using the secret" />
        <div className="flex max-w-lg flex-col gap-4">
          <div>
            Use placeholders in your files — Docker Compose, Kubernetes
            manifests, anything.
          </div>
          <WindowChrome filename="docker-compose.yml" className="h-full">
            <div>
              <YamlLine
                content={
                  <>
                    <YamlKey k="services" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={1}
                content={
                  <>
                    <YamlKey k="api" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={2}
                content={
                  <>
                    <YamlKey k="image" />
                    <span className="text-muted-foreground">: </span>
                    <YamlString v="myapp:latest" />
                  </>
                }
              />
              <YamlLine
                indent={2}
                content={
                  <>
                    <YamlKey k="environment" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={3}
                highlight
                content={
                  <>
                    <YamlKey k="API_KEY" />
                    <span className="text-muted-foreground">: </span>
                    <YamlPlaceholder v="__API_KEY__" />
                  </>
                }
              />
            </div>
          </WindowChrome>
          <div>
            The <span>__API_KEY__</span> placeholder gets replaced with the real
            value at deploy time.
          </div>
        </div>
      </div>

      <Connector direction="down" />

      {/* ─── PANEL 3: Inject and deploy ─── */}
      <div
        className={cn(isVisible ? "animate-fade-up" : "opacity-0")}
        style={{ animationDelay: "0.3s" }}
      >
        <StepLabel number="03" title="Inject and deploy" />
        <div className="flex flex-col gap-4">
          <div>
            Pipe through <code className="text-terminal">obscuro inject</code>{" "}
            and deploy
          </div>
          {/* Command */}
          <WindowChrome filename="terminal">
            <TerminalLine command="obscuro inject < docker-compose.yml | docker compose -f - up" />
          </WindowChrome>

          {/* Arrow connector (desktop only) */}
          <div className="hidden items-center self-center md:flex">
            <ArrowRight className="text-terminal/30 size-5" />
          </div>
          <Connector direction="down" />

          <div>This is what Docker Compose sees</div>
          {/* What Docker Compose sees */}
          <WindowChrome filename="stdin → docker compose">
            <div>
              <YamlLine
                content={
                  <>
                    <YamlKey k="services" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={1}
                content={
                  <>
                    <YamlKey k="api" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={2}
                content={
                  <>
                    <YamlKey k="image" />
                    <span className="text-muted-foreground">: </span>
                    <YamlString v="myapp:latest" />
                  </>
                }
              />
              <YamlLine
                indent={2}
                content={
                  <>
                    <YamlKey k="environment" />
                    <span className="text-muted-foreground">:</span>
                  </>
                }
              />
              <YamlLine
                indent={3}
                highlight
                content={
                  <>
                    <YamlKey k="API_KEY" />
                    <span className="text-muted-foreground">: </span>
                    <YamlString v="sk-live-abc123" />
                  </>
                }
              />
            </div>
            <div className="text-muted-foreground/50 mt-3 flex items-center gap-1.5 text-[11px]">
              <Play className="size-3" />
              Secrets injected at runtime — never written to disk
            </div>
          </WindowChrome>
        </div>
      </div>
    </div>
  )
}
