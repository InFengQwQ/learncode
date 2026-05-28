import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: string
  title: string
  description?: string
  action?: ReactNode
}

export default function EmptyState({
  icon,
  title,
  description,
  action,
}: EmptyStateProps) {
  return (
    <div className="flex animate-fade-in-up flex-col items-center justify-center py-20 text-center">
      {icon && (
        <span className="mb-5 text-5xl opacity-60" aria-hidden="true">
          {icon}
        </span>
      )}
      <h3 className="text-lg font-medium text-text-primary">
        {title}
      </h3>
      {description && (
        <p className="mt-2 max-w-md text-sm text-text-secondary">
          {description}
        </p>
      )}
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}
