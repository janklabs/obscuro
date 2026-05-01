import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  `
    group/button inline-flex shrink-0 items-center justify-center rounded-lg
    border border-transparent bg-clip-padding text-sm font-medium
    whitespace-nowrap transition-all outline-none select-none
    focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50
    active:not-aria-[haspopup]:translate-y-px
    disabled:pointer-events-none disabled:opacity-50
    aria-invalid:border-destructive aria-invalid:ring-3
    aria-invalid:ring-destructive/20
    dark:aria-invalid:border-destructive/50
    dark:aria-invalid:ring-destructive/40
    [&_svg]:pointer-events-none [&_svg]:shrink-0
    [&_svg:not([class*='size-'])]:size-4
  `,
  {
    variants: {
      variant: {
        default: `
            bg-primary text-primary-foreground transition-all duration-200
            hover:bg-[#d08ff7] hover:shadow-[0_0_16px_rgba(191,90,242,0.25)]
            [a]:hover:bg-primary/80
          `,
        outline: `
            border-border bg-background transition-all duration-200
            hover:border-terminal/40 hover:bg-muted hover:text-foreground
            hover:shadow-[0_0_12px_rgba(191,90,242,0.1)]
            aria-expanded:bg-muted aria-expanded:text-foreground
            dark:border-input dark:bg-input/30
            dark:hover:bg-input/50
          `,
        secondary: `
            bg-secondary text-secondary-foreground transition-all duration-200
            hover:bg-secondary/80 hover:shadow-[0_0_12px_rgba(191,90,242,0.1)]
            aria-expanded:bg-secondary aria-expanded:text-secondary-foreground
          `,
        ghost: `
            transition-all duration-200
            hover:bg-terminal/10 hover:text-foreground
            aria-expanded:bg-muted aria-expanded:text-foreground
            dark:hover:bg-terminal/10
          `,
        destructive: `
            bg-destructive/10 text-destructive
            hover:bg-destructive/20
            focus-visible:border-destructive/40
            focus-visible:ring-destructive/20
            dark:bg-destructive/20
            dark:hover:bg-destructive/30
            dark:focus-visible:ring-destructive/40
          `,
        link: `
          text-primary underline-offset-4 transition-colors duration-200
          hover:text-[#d08ff7] hover:underline
        `,
        glow: `
          bg-terminal font-bold tracking-wider text-[#050505] uppercase
          transition-all duration-300
          hover:bg-[#d08ff7]
          hover:shadow-[0_0_24px_rgba(191,90,242,0.4),0_0_48px_rgba(191,90,242,0.15)]
        `,
        "ghost-glow": `
            border-border font-bold tracking-wider text-muted-foreground
            uppercase transition-all duration-200
            hover:border-terminal/50 hover:text-terminal
            hover:shadow-[0_0_12px_rgba(191,90,242,0.15)]
          `,
      },
      size: {
        default: `
            h-8 gap-1.5 px-2.5
            has-data-[icon=inline-end]:pr-2
            has-data-[icon=inline-start]:pl-2
          `,
        xs: `
          h-6 gap-1 rounded-[min(var(--radius-md),10px)] px-2 text-xs
          in-data-[slot=button-group]:rounded-lg
          has-data-[icon=inline-end]:pr-1.5
          has-data-[icon=inline-start]:pl-1.5
          [&_svg:not([class*='size-'])]:size-3
        `,
        sm: `
          h-7 gap-1 rounded-[min(var(--radius-md),12px)] px-2.5 text-[0.8rem]
          in-data-[slot=button-group]:rounded-lg
          has-data-[icon=inline-end]:pr-1.5
          has-data-[icon=inline-start]:pl-1.5
          [&_svg:not([class*='size-'])]:size-3.5
        `,
        lg: `
          h-9 gap-1.5 px-2.5
          has-data-[icon=inline-end]:pr-2
          has-data-[icon=inline-start]:pl-2
        `,
        icon: "size-8",
        "icon-xs": `
            size-6 rounded-[min(var(--radius-md),10px)]
            in-data-[slot=button-group]:rounded-lg
            [&_svg:not([class*='size-'])]:size-3
          `,
        "icon-sm": `
            size-7 rounded-[min(var(--radius-md),12px)]
            in-data-[slot=button-group]:rounded-lg
          `,
        "icon-lg": "size-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
