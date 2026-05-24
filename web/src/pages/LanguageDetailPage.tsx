import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { languages, versions, type Language, type LanguageVersion } from '../api/client'
import { formatDate, externalURL, isIconURL } from '../lib/utils'
import Card from '../components/ui/Card'
import Badge from '../components/ui/Badge'
import Button from '../components/ui/Button'

export default function LanguageDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [lang, setLang] = useState<Language | null>(null)
  const [vers, setVers] = useState<LanguageVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (!id) return
    Promise.all([
      languages.get(id),
      versions.listByLanguage(id),
    ]).then(([langRes, verRes]) => {
      if (langRes.ok && langRes.data) {
        setLang(langRes.data)
        if (verRes.ok && verRes.data) {
          setVers(verRes.data)
        }
      } else {
        setError(langRes.error ?? '加载失败')
      }
      setLoading(false)
    })
  }, [id])

  async function handleDelete() {
    if (!id || !lang) return
    if (!confirm(`确定要删除 "${lang.name}"？此操作不可撤销。`)) return
    setDeleting(true)
    try {
      const res = await languages.delete(id)
      if (res.ok) {
        navigate('/languages')
      } else {
        setError(res.error ?? '删除失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-sm text-stone-500">加载中...</p>
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

  if (!lang) return null

  return (
    <div>
      <Link to="/languages" className="text-sm text-stone-500 transition-colors hover:text-stone-300">
        &larr; 语言列表
      </Link>

      <Card className="mt-6 overflow-hidden">
        {/* Header area */}
        <div className="flex items-start justify-between gap-6 bg-stone-800/50 px-6 py-5">
          <div className="flex items-center gap-4 min-w-0">
            {lang.icon && isIconURL(lang.icon) ? (
              <img src={lang.icon} alt="" className="h-12 w-12 shrink-0 rounded-lg object-contain" />
            ) : (
              <span className="text-4xl shrink-0">{lang.icon || '📄'}</span>
            )}
            <div className="min-w-0">
              <h1 className="text-2xl font-bold truncate">{lang.name}</h1>
              <div className="mt-1.5 flex items-center gap-2 text-sm text-stone-400">
                <code className="rounded bg-stone-800 px-1.5 py-0.5 font-mono text-xs">
                  {lang.slug}
                </code>
                <span>·</span>
                <Badge
                  variant={lang.compatibility_model === 'strict' ? 'info' : 'warning'}
                >
                  {lang.compatibility_model}
                </Badge>
              </div>
            </div>
          </div>
          <Button
            variant="danger"
            size="sm"
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? '删除中...' : '删除'}
          </Button>
        </div>

        {/* Detail body */}
        <div className="px-6 py-5 space-y-4">
          {lang.source_urls?.docs && (
            <div>
              <p className="text-xs font-medium text-stone-500 uppercase tracking-wide">
                官方文档
              </p>
              <a
                href={externalURL(lang.source_urls.docs)}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-1 inline-block text-sm text-amber-500 break-all transition-colors hover:text-amber-400"
              >
                {lang.source_urls.docs}
              </a>
            </div>
          )}
          {lang.source_urls?.runtime && (
            <div>
              <p className="text-xs font-medium text-stone-500 uppercase tracking-wide">
                运行时下载
              </p>
              <a
                href={externalURL(lang.source_urls.runtime)}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-1 inline-block text-sm text-amber-500 break-all transition-colors hover:text-amber-400"
              >
                {lang.source_urls.runtime}
              </a>
            </div>
          )}
          <div>
            <p className="text-xs font-medium text-stone-500 uppercase tracking-wide">
              创建时间
            </p>
            <p className="mt-1 text-sm text-stone-400">
              {formatDate(lang.created_at)}
            </p>
          </div>
        </div>
      </Card>

      {/* Versions section */}
      <div className="mt-10">
        <h2 className="text-lg font-semibold">语言版本</h2>
        {vers.length === 0 ? (
          <div className="mt-4 rounded-xl border border-stone-800 bg-stone-900 px-6 py-8 text-center">
            <p className="text-sm text-stone-400">尚未配置任何版本。</p>
            <p className="mt-2 text-sm text-stone-500">
              版本定义了该语言的运行时环境（容器镜像、编译器版本等），
              后续将在此处添加和管理版本。
            </p>
          </div>
        ) : (
          <div className="mt-4 divide-y divide-stone-800 rounded-xl border border-stone-800">
            {vers.map((v) => (
              <div key={v.id} className="flex items-center justify-between px-6 py-4">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-sm">{v.version}</span>
                  {v.initialized && (
                    <Badge variant="success">就绪</Badge>
                  )}
                </div>
                <span className="text-xs text-stone-500">{v.status}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
