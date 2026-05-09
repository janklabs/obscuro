"use client"

import Link from "next/link"
import { ArrowDown, ArrowRight } from "lucide-react"

import { buttonVariants } from "@/components/ui/button"
import { CodeBlock } from "./_components/code-block"
import { FeatureGrid } from "./_components/feature-grid"
import { Footer } from "./_components/footer"
import { HowItWorks } from "./_components/how-it-works"
import { Nav } from "./_components/nav"
import { SectionHeading } from "./_components/section-heading"
import { TerminalAnimation } from "./_components/terminal-animation"
import Waves from "./_components/waves"

const INSTALL_SCRIPT =
  "curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | bash"

export default function Home() {
  return (
    <>
      <Nav />
      <main>
        {/* ─── Section 1: Hero with Waves ─── */}
        <section
          className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden"
          aria-labelledby="hero-heading"
        >
          {/* Waves background */}
          <div className="pointer-events-none absolute inset-0">
            <Waves
              lineColor="rgba(191, 90, 242, 0.12)"
              backgroundColor="transparent"
              waveSpeedX={0.02}
              waveSpeedY={0.01}
              waveAmpX={40}
              waveAmpY={20}
              friction={0.9}
              tension={0.01}
              maxCursorMove={120}
              xGap={12}
              yGap={36}
            />
          </div>

          {/* Content */}
          <div className="relative z-10 flex flex-col items-center px-6 text-center">
            <h1
              id="hero-heading"
              className="leading-[1.1] font-bold tracking-tight"
              style={{ fontFamily: "var(--font-display)" }}
            >
              <span className="animate-fade-up text-foreground block text-3xl md:text-5xl lg:text-6xl">
                Encrypt secrets.
              </span>
              <span
                className="animate-fade-up text-foreground block text-3xl md:text-5xl lg:text-6xl"
                style={{ animationDelay: "0.15s" }}
              >
                Commit safely.
              </span>
              <span
                className="animate-fade-up text-glow text-terminal block text-3xl md:text-5xl lg:text-6xl"
                style={{ animationDelay: "0.3s" }}
              >
                Deploy anywhere.
              </span>
            </h1>

            <p
              className="animate-fade-up text-muted-foreground mt-5 max-w-md text-base md:text-lg"
              style={{ animationDelay: "0.4s" }}
            >
              A CLI that encrypts secrets in your git repo and injects them at
              deploy time.
            </p>

            <div
              className="animate-fade-up mt-12"
              style={{ animationDelay: "0.6s" }}
            >
              <div className="mx-auto max-w-[672px] min-w-[672px]">
                <TerminalAnimation />
              </div>
            </div>

            <div
              className="animate-fade-up mt-10"
              style={{ animationDelay: "0.9s" }}
            >
              <Link
                href="/docs"
                className={buttonVariants({
                  variant: "glow",
                  size: "lg",
                  className: "gap-2",
                })}
              >
                Install Obscuro
                <ArrowRight className="size-3.5" />
              </Link>
            </div>
          </div>

          {/* Down arrow */}
          <a
            href="#how-it-works"
            className="animate-fade-up border-border bg-card/50 text-muted-foreground hover:border-terminal hover:text-terminal absolute bottom-10 z-10 flex size-10 items-center justify-center rounded-full border backdrop-blur-sm transition-all duration-300 hover:shadow-[0_0_20px_var(--color-terminal-dim)]"
            style={{ animationDelay: "1.3s" }}
            aria-label="Scroll to how it works"
          >
            <ArrowDown className="size-4" />
          </a>
        </section>

        {/* ─── Section 2: How It Works (no waves) ─── */}
        <section
          id="how-it-works"
          className="border-border bg-background relative z-10 border-t px-20 py-24"
          aria-labelledby="how-it-works-heading"
        >
          <div className="mx-auto">
            <SectionHeading id="how-it-works-heading">
              How it works
            </SectionHeading>
            <HowItWorks />
          </div>
        </section>

        {/* ─── Section 3: Features ─── */}
        <section
          className="border-border bg-background relative z-10 border-t px-6 py-24"
          aria-labelledby="features-heading"
        >
          <div className="mx-auto max-w-5xl">
            <SectionHeading id="features-heading">Features</SectionHeading>
            <FeatureGrid />
          </div>
        </section>

        {/* ─── Section 4: Install + CTAs ─── */}
        <section
          className="border-border bg-background relative z-10 border-t px-6 py-24"
          aria-labelledby="install-heading"
        >
          <div className="mx-auto flex max-w-5xl flex-col items-center text-center">
            <SectionHeading id="install-heading">Get started</SectionHeading>
            <div className="flex flex-col items-center gap-4">
              <div>Here&apos;s the install script.</div>
              <CodeBlock code={INSTALL_SCRIPT} className="" />
              <div>Or read the docs first</div>
              <div className="flex items-center gap-4">
                <Link
                  href="/docs"
                  className={buttonVariants({
                    variant: "glow",
                    size: "lg",
                    className: "gap-2",
                  })}
                >
                  Read the Docs
                  <ArrowRight className="size-3.5" />
                </Link>
                <a
                  href="https://github.com/janklabs/obscuro"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={buttonVariants({
                    variant: "ghost-glow",
                    size: "lg",
                  })}
                >
                  GitHub
                </a>
              </div>
            </div>
          </div>
        </section>

        {/* Footer */}
        <Footer />
      </main>
    </>
  )
}
