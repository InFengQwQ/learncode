import Button from '../components/ui/Button'

export default function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <p className="animate-fade-in font-mono text-8xl font-bold text-bg-subtle">
        404
      </p>
      <h1 className="animate-fade-in-up mt-4 text-xl font-semibold text-text-primary">
        页面未找到
      </h1>
      <p
        className="mt-2 text-sm text-text-secondary"
        style={{ animationDelay: '150ms' }}
      >
        你访问的页面不存在。
      </p>
      <div
        className="mt-8"
        style={{ animationDelay: '300ms' }}
      >
        <Button href="/" variant="ghost" size="md">
          &larr; 返回首页
        </Button>
      </div>
    </div>
  )
}
