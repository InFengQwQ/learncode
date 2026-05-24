import Button from '../components/ui/Button'

export default function HomePage() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <h1 className="max-w-4xl text-5xl font-bold leading-tight tracking-tight text-pretty">
        掌握编程，
        <br />
        <span className="text-amber-500">一门语言接一门语言</span>
      </h1>
      <p className="mt-6 max-w-xl text-lg leading-relaxed text-stone-400">
        LearnCode 为每一门编程语言构建个性化学习路径。
        动态练习、实时验证、基于你进度的知识库——全部自动生成。
      </p>
      <div className="mt-10 flex gap-4">
        <Button href="/languages" size="lg">
          开始学习
        </Button>
        <Button href="/languages/add" variant="secondary" size="lg">
          添加一门语言
        </Button>
      </div>
    </div>
  )
}
