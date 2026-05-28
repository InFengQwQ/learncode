import type { CSSProperties, ReactNode } from 'react'

interface CardProps {
  children: ReactNode
  hover?: boolean
  depth?: 'flat' | 'raised' | 'elevated'
  className?: string
  style?: CSSProperties
}

const depthClasses = {
  flat: '',
  raised: 'shadow-lg shadow-black/20',
  elevated: 'shadow-xl shadow-black/30',
}

export default function Card({ children, hover = false, depth = 'flat', className = '' }: CardProps) {
  return (
    <div
      className={`rounded-xl border border-border bg-bg-elevated ${depthClasses[depth]} ${
        hover ? 'transition-all duration-200 hover:border-border-hover hover:bg-bg-hover' : ''
      } ${className}`}
    >
      {children}
    </div>
  )
}
