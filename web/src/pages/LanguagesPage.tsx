import { Link } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import { languages, type Language } from '../api/client'
import { formatDate, isIconURL } from '../lib/utils'
import Card from '../components/ui/Card'
import Badge from '../components/ui/Badge'
import Button from '../components/ui/Button'
import EmptyState from '../components/ui/EmptyState'
import Skeleton from '../components/ui/Skeleton'

function LanguageCardSkeleton() {
  return (
    <div className="rounded-xl border border-border bg-bg-elevated p-6">
      <div className="flex items-center gap-3">
        <Skeleton className="h-8 w-8 rounded-lg" />
        <div className="min-w-0 flex-1">
          <Skeleton className="mb-1.5 h-5 w-28" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
      <div className="mt-4 flex items-center gap-2">
        <Skeleton className="h-5 w-16 rounded-full" />
        <Skeleton className="h-5 w-14 rounded-full" />
      </div>
    </div>
  )
}

export default function LanguagesPage() {
  const { data, loading, error } = useApi<Language[]>(
    () => languages.list(),
    { fetchOnMount: true },
  )

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">编程语言</h1>
          <p className="mt-1 text-sm text-text-secondary">
            管理已验证的编程语言知识库
          </p>
        </div>
        <Button href="/languages/add" size="md">
          添加语言
        </Button>
      </div>

      {loading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <LanguageCardSkeleton key={i} />
          ))}
        </div>
      )}

      {error && (
        <div className="flex items-center justify-center py-20">
          <p className="text-sm text-danger">错误: {error}</p>
        </div>
      )}

      {!loading && !error && (data?.length ?? 0) === 0 && (
        <EmptyState
          icon="{}"
          title="还没有语言"
          description="添加第一门编程语言来开始构建知识库"
          action={<Button href="/languages/add">添加第一门语言</Button>}
        />
      )}

      {!loading && !error && data && data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((lang, idx) => (
            <Link key={lang.id} to={`/languages/${lang.id}`}>
              <Card
                hover
                className={`h-full p-6 transition-all duration-200 ${
                  lang.status !== 'active'
                    ? 'opacity-70 ring-1 ring-accent-muted/30'
                    : ''
                }`}
                style={{ animationDelay: `${idx * 60}ms` }}
              >
                <div className="flex items-center gap-3">
                  {lang.icon && isIconURL(lang.icon) ? (
                    <img
                      src={lang.icon}
                      alt=""
                      width={32}
                      height={32}
                      className="h-8 w-8 shrink-0 rounded-lg object-contain"
                    />
                  ) : (
                    <span className="text-2xl" aria-hidden="true">
                      {lang.icon || '📄'}
                    </span>
                  )}
                  <div className="min-w-0">
                    <h2 className="truncate text-lg font-semibold text-text-primary">
                      {lang.name}
                    </h2>
                    <p className="text-sm text-text-muted">{lang.slug}</p>
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
                  <Badge
                    variant={lang.status === 'active' ? 'success' : 'danger'}
                  >
                    {lang.status === 'active' ? '已激活' : '待构建'}
                  </Badge>
                  <span className="text-xs text-text-muted">
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
