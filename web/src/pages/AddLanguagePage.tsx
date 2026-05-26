import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { languages, type InitSuggestion } from '../api/client'
import { isIconURL, friendlyError } from '../lib/utils'
import Card from '../components/ui/Card'
import Badge from '../components/ui/Badge'
import Button from '../components/ui/Button'

export default function AddLanguagePage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [querying, setQuerying] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState('')
  const [suggestion, setSuggestion] = useState<InitSuggestion | null>(null)

  async function handleQuery() {
    if (!name.trim()) return
    setQuerying(true)
    setError('')
    setSuggestion(null)
    try {
      const res = await languages.initQuery(name.trim())
      if (res.ok && res.data) {
        setSuggestion(res.data)
      } else {
        setError(friendlyError(res.error ?? '查询失败'))
      }
    } catch {
      setError('网络错误')
    } finally {
      setQuerying(false)
    }
  }

  function handleCancel() {
    setSuggestion(null)
  }

  async function handleConfirm() {
    if (!suggestion) return
    setConfirming(true)
    setError('')
    try {
      // Create with ALL discovered versions — user manages them later from detail page
      const allVersions = suggestion.versions?.map(v => v.version) ?? []
      const res = await languages.initConfirm({
        name: suggestion.name,
        slug: suggestion.slug,
        icon: suggestion.icon,
        compatibility_model: suggestion.compatibility_model,
        docs_url: '',
        runtime_url: '',
        versions: allVersions,
        discovered_versions: suggestion.versions,
      })
      if (res.ok && res.data) {
        navigate(`/languages/${res.data.language.id}`)
      } else {
        setError(res.error ?? '确认失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setConfirming(false)
    }
  }

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="text-2xl font-bold">添加语言</h1>
      <p className="mt-2 text-sm text-stone-400">
        输入编程语言名称。我们将通过 Wikipedia 查找并验证其信息。
      </p>

      {/* Search input */}
      <div className="mt-8 flex gap-3">
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleQuery()}
          placeholder="例如: python, rust, go…"
          className="flex-1 rounded-lg border border-stone-700 bg-stone-900 px-4 py-2.5 text-sm text-stone-100 placeholder-stone-500 transition-colors focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500"
        />
        <Button
          onClick={handleQuery}
          disabled={querying || !name.trim()}
        >
          {querying ? '搜索中…' : '查找语言'}
        </Button>
      </div>

      {error && (
        <p className="mt-4 text-sm text-red-400">{error}</p>
      )}

      {/* Read-only confirmation card */}
      {suggestion && (
        <Card className="mt-8 overflow-hidden">
          {/* Header */}
          <div className="flex items-center gap-4 bg-stone-800/50 px-6 py-5">
            {suggestion.icon && isIconURL(suggestion.icon) ? (
              <img src={suggestion.icon} alt="" width={48} height={48} className="h-12 w-12 shrink-0 rounded-lg object-contain" />
            ) : (
              <span className="text-4xl">{suggestion.icon || '📄'}</span>
            )}
            <div className="min-w-0">
              <h2 className="text-xl font-bold">{suggestion.name}</h2>
              <div className="mt-1.5 flex items-center gap-2 text-sm text-stone-400">
                <code className="rounded bg-stone-800 px-1.5 py-0.5 font-mono text-xs">
                  {suggestion.slug}
                </code>
                <span>·</span>
                <Badge
                  variant={
                    suggestion.compatibility_model === 'strict'
                      ? 'info'
                      : 'warning'
                  }
                >
                  {suggestion.compatibility_model}
                </Badge>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="px-6 py-5 space-y-4">
            <p className="text-sm leading-relaxed text-stone-300">
              {suggestion.description}
            </p>

            {/* Versions preview — all discovered versions will be created */}
            {suggestion.versions && suggestion.versions.length > 0 && (
              <div>
                <p className="text-xs font-medium text-stone-500 uppercase tracking-wide mb-2">
                  收录版本 ({suggestion.versions.length})
                </p>
                <div className="space-y-2">
                  {suggestion.versions.map((v) => (
                    <div key={v.version} className="flex items-center gap-3 rounded-lg border border-stone-700 bg-stone-900 px-4 py-3">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm font-medium">{v.version}</span>
                          {v.lts && (<Badge variant="success">LTS</Badge>)}
                          {v.version === suggestion.latest_version && (<Badge variant="info">最新</Badge>)}
                        </div>
                        <p className="mt-0.5 text-xs text-stone-500 truncate">{v.brief}</p>
                      </div>
                      <span className="text-xs text-stone-600 shrink-0">{v.released}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex gap-3 px-6 py-4 border-t border-stone-800">
            <Button
              onClick={handleConfirm}
              disabled={confirming}
              className="flex-1"
            >
              {confirming ? '创建中…' : '确认添加'}
            </Button>
            <Button
              onClick={handleCancel}
              variant="secondary"
              disabled={confirming}
              className="flex-1"
            >
              取消
            </Button>
          </div>
        </Card>
      )}
    </div>
  )
}
