import { cn } from "@/lib/utils"

interface SectionHeadingProps {
  children: React.ReactNode
  className?: string
  id?: string
}

export function SectionHeading({
  children,
  className,
  id,
}: SectionHeadingProps) {
  return (
    <h2
      id={id}
      className={cn(
        "text-muted-foreground mb-8 text-xs font-bold tracking-[0.3em] uppercase",
        className,
      )}
    >
      {children}
    </h2>
  )
}
