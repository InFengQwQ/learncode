import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { languages, type Language } from '../api/client'
import { formatDate, isIconURL } from '../lib/utils'
import Card from '../components/ui/Card'
import Badge from '../components/ui/Badge'
import Button from '../components/ui/Button'
import EmptyState from '../components/ui/EmptyState'

export default function LanguagesPage() {
  const [data, setData] = useState<Language[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    languages.list().then((res) => {
  	if (res.ok) {
        	setData(res.data ?? [])
      } else {
        setError(res.error ?? '加载失败')
      }
      setLoading(false)
    })
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-sm text-stone-500">加载中…</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-sm text-red-400">错误: {error}</p>
      </div>
    )
  }

  return (
    <div>
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">编程语言</h1>
          <p className="mt-1 text-sm text-stone-400">
            管理已验证的编程语言知识库
          </p>
        </div>
        <Button href="/languages/add" size="md">
          添加语言
        </Button>
      </div>

      {data.length === 0 ? (
        <EmptyState
          icon="{}"
          title="还没有语言"
          description="添加第一门编程语言来开始构建知识库"
          action={<Button href="/languages/add">添加第一门语言</Button>}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((lang) => (
            <Link key={lang.id} to={`/languages/${lang.id}`}>
              <Card hover className={`p-6 h-full ${lang.status !== 'active' ? 'opacity-70 ring-1 ring-amber-800/30' : ''}`}>
                <div className="flex items-center gap-3">
                  {lang.icon && isIconURL(lang.icon) ? (
                    <img src={lang.icon} alt="" width={32} height={32} className="h-8 w-8 shrink-0 rounded-lg object-contain" />
                  ) : (
                    <span className="text-2xl">{lang.icon || '📄'}</span>
                  )}
                  <div className="min-w-0">
                    <h2 className="text-lg font-semibold truncate">{lang.name}</h2>
                    <p className="text-sm text-stone-500">{lang.slug}</p>
                  </div>
                </div>
                <div className="mt-4 flex items-center gap-2">
                  <Badge
                    variant={
                      lang.compatibility_model === 'strict'
                        ? 'info'
                        : lang.compatibility_model === 'versioned'
                          ? 'warning'
                          : 'default'
                    }
                  >
                    {lang.compatibility_model}
                  </Badge>
                  <Badge variant={lang.status === 'active' ? 'success' : 'danger'}>
                    {lang.status === 'active' ? '已激活' : '待构建'}
                  </Badge>
                  <span className="text-xs text-stone-600">
                    {formatDate(lang.created_at)}
                  </span>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
