import { Link } from 'react-router-dom'
import type { ButtonHTMLAttributes, ReactNode } from 'react'

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  href?: string
  children: ReactNode
}

const variantClasses: Record<Variant, string> = {
  primary:
    'bg-amber-500 text-stone-900 hover:bg-amber-400 focus:ring-amber-500',
  secondary:
    'border border-stone-700 text-stone-200 hover:border-stone-500 hover:text-stone-100 focus:ring-stone-500',
  danger:
    'border border-red-800/50 text-red-400 hover:border-red-700 hover:text-red-300 focus:ring-red-500',
  ghost:
    'text-stone-400 hover:text-stone-200 focus:ring-stone-500',
}

const sizeClasses: Record<Size, string> = {
  sm: 'px-3 py-1.5 text-xs rounded-md',
  md: 'px-4 py-2 text-sm rounded-lg',
  lg: 'px-6 py-3 text-sm rounded-lg',
}

export default function Button({
  variant = 'primary',
  size = 'md',
  href,
  children,
  className = '',
  disabled,
  ...rest
}: ButtonProps) {
  const classes = `inline-flex items-center justify-center font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-stone-950 disabled:opacity-50 disabled:pointer-events-none ${variantClasses[variant]} ${sizeClasses[size]} ${className}`

  if (href) {
    return (
      <Link to={href} className={classes}>
        {children}
      </Link>
    )
  }

  return (
    <button className={classes} disabled={disabled} {...rest}>
      {children}
    </button>
  )
}
