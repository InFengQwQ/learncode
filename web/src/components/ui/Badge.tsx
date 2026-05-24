import type { ReactNode } from 'react'

type Variant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface BadgeProps {
  variant?: Variant
  children: ReactNode
  className?: string
}

const variantClasses: Record<Variant, string> = {
  default: 'bg-stone-800 text-stone-300',
  success: 'bg-emerald-900/50 text-emerald-400',
  warning: 'bg-amber-900/50 text-amber-300',
  danger: 'bg-red-900/50 text-red-400',
  info: 'bg-blue-900/50 text-blue-300',
}

export default function Badge({
  variant = 'default',
  children,
  className = '',
}: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${variantClasses[variant]} ${className}`}
    >
      {children}
    </span>
  )
}
