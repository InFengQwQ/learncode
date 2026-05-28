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
    'bg-accent text-bg-base hover:bg-accent-hover active:scale-[0.97] focus:ring-accent',
  secondary:
    'border border-border text-text-primary hover:border-border-hover hover:text-text-primary active:scale-[0.97] focus:ring-accent',
  danger:
    'border border-danger-bg text-danger hover:border-danger hover:text-danger active:scale-[0.97] focus:ring-danger',
  ghost:
    'text-text-secondary hover:text-text-primary active:scale-[0.97] focus:ring-accent',
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
  const classes = `inline-flex items-center justify-center font-medium transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-bg-base disabled:opacity-50 disabled:pointer-events-none disabled:scale-100 ${variantClasses[variant]} ${sizeClasses[size]} ${className}`

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
