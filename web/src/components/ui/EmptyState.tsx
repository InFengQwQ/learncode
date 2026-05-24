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
    <div className="flex flex-col items-center justify-center py-20 text-center">
      {icon && <span className="text-5xl">{icon}</span>}
      <h3 className={`${icon ? 'mt-5' : ''} text-lg font-medium text-stone-300`}>
        {title}
      </h3>
      {description && (
        <p className="mt-2 max-w-md text-sm text-stone-500">{description}</p>
      )}
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}
