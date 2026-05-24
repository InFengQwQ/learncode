import Button from '../components/ui/Button'

export default function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <p className="font-mono text-8xl font-bold text-stone-800">404</p>
      <h1 className="mt-4 text-xl font-semibold text-stone-300">页面未找到</h1>
      <p className="mt-2 text-sm text-stone-500">
        你访问的页面不存在。
      </p>
      <div className="mt-8">
        <Button href="/" variant="ghost" size="md">
          &larr; 返回首页
        </Button>
      </div>
    </div>
  )
}
