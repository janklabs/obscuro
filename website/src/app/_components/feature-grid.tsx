"use client"

import {
  ArrowUpDown,
  FolderGit2,
  KeyRound,
  Lock,
  Pencil,
  Terminal,
} from "lucide-react"

import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll"
import { GlowCard } from "./glow-card"

const features = [
  {
    title: "Encrypted at rest",
    description: "Secrets are encrypted before they touch disk.",
    icon: Lock,
  },
  {
    title: "Helm-native",
    description: "One flag to inject secrets at deploy time.",
    icon: Terminal,
  },
  {
    title: "OS Keychain",
    description:
      "Master password lives in your system keychain. No env vars needed.",
    icon: KeyRound,
  },
  {
    title: "Lives in git",
    description:
      "Secrets version-controlled alongside your code. Portable and auditable.",
    icon: FolderGit2,
  },
  {
    title: "Edit in $EDITOR",
    description: "Opens in your editor. Re-encrypts on save.",
    icon: Pencil,
  },
  {
    title: "Self-updating",
    description: "obscuro upgrade. That's it.",
    icon: ArrowUpDown,
  },
]

export function FeatureGrid() {
  const { ref, isVisible } = useAnimateOnScroll()

  return (
    <div
      ref={ref}
      className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3"
    >
      {features.map((feature, i) => (
        <GlowCard
          key={feature.title}
          className={isVisible ? "animate-fade-up" : "opacity-0"}
          style={{ animationDelay: `${i * 0.1}s` }}
        >
          <div className="mb-3 flex items-center gap-3">
            <feature.icon className="text-muted-foreground group-hover:text-terminal size-5 transition-colors duration-300" />
            <h3 className="text-foreground text-sm font-bold tracking-wide">
              {feature.title}
            </h3>
          </div>
          <p className="text-muted-foreground text-xs/relaxed">
            {feature.description}
          </p>
        </GlowCard>
      ))}
    </div>
  )
}
