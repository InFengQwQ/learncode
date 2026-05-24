import type { ReactNode } from 'react'

interface CardProps {
  children: ReactNode
  hover?: boolean
  className?: string
}

export default function Card({ children, hover = false, className = '' }: CardProps) {
  return (
    <div
      className={`rounded-xl border border-stone-800 bg-stone-900 ${hover ? 'transition-colors hover:border-stone-600' : ''} ${className}`}
    >
      {children}
    </div>
  )
}
