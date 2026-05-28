interface SkeletonProps {
  className?: string
}

export default function Skeleton({ className = '' }: SkeletonProps) {
  return (
    <div
      className={`animate-shimmer rounded-lg bg-bg-subtle ${className}`}
      style={{
        backgroundImage: 'linear-gradient(90deg, var(--color-bg-subtle) 0%, var(--color-bg-hover) 40%, var(--color-bg-subtle) 80%)',
        backgroundSize: '200% 100%',
      }}
      aria-hidden="true"
    />
  )
}
