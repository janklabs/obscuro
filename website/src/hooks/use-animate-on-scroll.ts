"use client"

import { useEffect, useRef, useState } from "react"

interface UseAnimateOnScrollOptions {
  /** Threshold for intersection (0-1). Default 0.1 */
  threshold?: number
  /** Root margin for early/late triggering. Default "0px" */
  rootMargin?: string
  /** Only trigger once. Default true */
  once?: boolean
}

export function useAnimateOnScroll({
  threshold = 0.1,
  rootMargin = "0px 0px -40px 0px",
  once = true,
}: UseAnimateOnScrollOptions = {}) {
  const ref = useRef<HTMLDivElement>(null)
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true)
          if (once) observer.unobserve(el)
        } else if (!once) {
          setIsVisible(false)
        }
      },
      { threshold, rootMargin },
    )

    observer.observe(el)
    return () => observer.disconnect()
  }, [threshold, rootMargin, once])

  return { ref, isVisible }
}
