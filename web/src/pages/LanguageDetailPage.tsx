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
  const [showAddVersion, setShowAddVersion] = useState(false)
  const [newVersion, setNewVersion] = useState('')
  const [addingVersion, setAddingVersion] = useState(false)

  function loadVersions() {
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
  }

  useEffect(() => {
    loadVersions()
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

  async function handleAddVersion() {
    if (!id || !newVersion.trim()) return
    setAddingVersion(true)
    try {
      const res = await versions.create(id, { version: newVersion.trim() })
      if (res.ok && res.data) {
        setVers([...vers, res.data])
        setNewVersion('')
        setShowAddVersion(false)
      } else {
        setError(res.error ?? '创建版本失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setAddingVersion(false)
    }
  }

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
              <img src={lang.icon} alt="" width={48} height={48} className="h-12 w-12 shrink-0 rounded-lg object-contain" />
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
            {deleting ? '删除中…' : '删除'}
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
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">语言版本</h2>
          <Button variant="ghost" size="sm" onClick={() => setShowAddVersion(!showAddVersion)}>
            + 添加版本
          </Button>
        </div>

        {/* Add version form */}
        {showAddVersion && (
          <div className="mt-4 flex gap-3">
            <input
              type="text"
              value={newVersion}
              onChange={(e) => setNewVersion(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAddVersion()}
              placeholder="例如: 3.12, 21, 1.22"
              className="flex-1 rounded-lg border border-stone-700 bg-stone-900 px-4 py-2.5 text-sm text-stone-100 placeholder-stone-500 transition-colors focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500"
            />
            <Button
              onClick={handleAddVersion}
              disabled={addingVersion || !newVersion.trim()}
            >
              {addingVersion ? '创建中…' : '创建'}
            </Button>
          </div>
        )}

        {vers.length === 0 ? (
          <div className="mt-4 rounded-xl border border-stone-800 bg-stone-900 px-6 py-8 text-center">
            <p className="text-sm text-stone-400">尚未配置任何版本。</p>
            <p className="mt-2 text-sm text-stone-500">
              添加一个版本后即可在 Playground 中编写和运行代码。
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
                <Link
                  to={`/playground/${id}?version=${v.id}`}
                  className="text-xs text-amber-500 transition-colors hover:text-amber-400"
                >
                  运行 &rarr;
                </Link>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
