import type { Language } from '../../api/client'
import { formatDate, isIconURL } from '../../lib/utils'
import Badge from '../ui/Badge'
import Button from '../ui/Button'

interface DetailHeaderProps {
  language: Language
  researching: boolean
  deleting: boolean
  onResearch: () => void
  onCancelResearch: () => void
  onDelete: () => void
}

export default function DetailHeader({
  language: lang,
  researching,
  deleting,
  onResearch,
  onCancelResearch,
  onDelete,
}: DetailHeaderProps) {
  return (
    <>
      {/* Inactive language warning */}
      {lang.status !== 'active' && (
        <div className="mb-6 rounded-xl border border-accent-muted/50 bg-accent-faint px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="text-2xl" aria-hidden="true">⚠️</span>
            <div>
              <p className="text-sm font-medium text-accent">
                知识库尚未构建 — 此语言当前不可用于学习
              </p>
              <p className="mt-1 text-xs text-accent/60">
                请先初始化版本环境，然后构建知识库。知识库完成后，语言将自动激活。
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Header card */}
      <div className="overflow-hidden rounded-xl border border-border bg-bg-elevated">
        <div className="flex items-start justify-between gap-6 bg-bg-hover/50 px-6 py-5">
          <div className="flex min-w-0 items-center gap-4">
            {lang.icon && isIconURL(lang.icon) ? (
              <img
                src={lang.icon}
                alt=""
                width={48}
                height={48}
                className="h-12 w-12 shrink-0 rounded-lg object-contain"
              />
            ) : (
              <span className="text-4xl shrink-0" aria-hidden="true">
                {lang.icon || '📄'}
              </span>
            )}
            <div className="min-w-0">
              <h1 className="truncate text-2xl font-bold">{lang.name}</h1>
              <div className="mt-1.5 flex items-center gap-2 text-sm text-text-secondary">
                <code className="rounded bg-bg-subtle px-1.5 py-0.5 font-mono text-xs">
                  {lang.slug}
                </code>
                <span>·</span>
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
                <span>·</span>
                <Badge variant={lang.status === 'active' ? 'success' : 'warning'}>
                  {lang.status === 'active' ? '已激活' : '未激活'}
                </Badge>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {researching ? (
              <Button variant="ghost" size="sm" onClick={onCancelResearch}>
                取消研究
              </Button>
            ) : (
              <Button variant="ghost" size="sm" onClick={onResearch}>
                研究资源
              </Button>
            )}
            <Button
              variant="danger"
              size="sm"
              onClick={onDelete}
              disabled={deleting}
            >
              {deleting ? '删除中…' : '删除'}
            </Button>
          </div>
        </div>

        {/* Detail body */}
        <div className="space-y-4 px-6 py-5">
          <div>
            <p className="text-xs font-medium tracking-wide uppercase text-text-muted">
              创建时间
            </p>
            <p className="mt-1 text-sm text-text-secondary">
              {formatDate(lang.created_at)}
            </p>
          </div>
          {lang.researched_at && (
            <div>
              <p className="text-xs font-medium tracking-wide uppercase text-text-muted">
                最近研究
              </p>
              <p className="mt-1 text-sm text-text-secondary">
                {formatDate(lang.researched_at)}
              </p>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
