import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { languages, versions as versionsApi, execute, knowledge, type Language, type LanguageVersion, type KnowledgeEntry, type VersionInitResult } from '../api/client'
import { formatDate, isIconURL } from '../lib/utils'
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

  // Inline playground state
  const [activeVersionId, setActiveVersionId] = useState('')
  const [code, setCode] = useState('')
  const [output, setOutput] = useState('')
  const [running, setRunning] = useState(false)

  // Version initialization state
  const [initializingId, setInitializingId] = useState('')
  const [initErrors, setInitErrors] = useState<Record<string, string>>({})

  // KB build state
  const [buildingId, setBuildingId] = useState('')

  // KB display state
  const [expandedKB, setExpandedKB] = useState<string | null>(null)
  const [kbEntries, setKBEntries] = useState<Record<string, { shared: KnowledgeEntry[]; private: KnowledgeEntry[] }>>({})
  const [loadingKB, setLoadingKB] = useState<string | null>(null)

  async function handleToggleStatus(versionId: string, currentStatus: string) {
    const newStatus = currentStatus === 'active' ? 'archived' : 'active'
    try {
      const res = await versionsApi.setStatus(versionId, newStatus)
      if (res.ok && res.data) {
        setVers((prev) => prev.map((v) => v.id === versionId ? res.data! : v))
      }
    } catch {
      // silently fail
    }
  }

  function loadVersions() {
    if (!id) return
    Promise.all([
      languages.get(id),
      versionsApi.listByLanguage(id),
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

  function openPlayground(versionId: string) {
    setActiveVersionId(versionId)
    setOutput('')
    if (!code) {
      setCode('')
    }
  }

  async function handleRun() {
    if (!activeVersionId || !code.trim()) return
    setRunning(true)
    setOutput('')
    try {
      const res = await execute.run({
        version_id: activeVersionId,
        code: code.trim(),
      })
      if (res.ok && res.data) {
        const { stdout, stderr, exit_code, duration_ms } = res.data
        let out = ''
        if (stdout) out += stdout
        if (stderr) out += (out ? '\n' : '') + stderr
        out += `\n\n── 退出码: ${exit_code}  耗时: ${duration_ms}ms`
        setOutput(out)
      } else {
        setOutput('执行失败: ' + (res.error ?? '未知错误'))
      }
    } catch {
      setOutput('网络错误')
    } finally {
      setRunning(false)
    }
  }

  async function handleInitialize(versionId: string) {
    setInitializingId(versionId)
    setInitErrors((prev) => { const next = { ...prev }; delete next[versionId]; return next })
    try {
      const res = await versionsApi.initialize(versionId)
      if (res.ok && res.data) {
        const initResult = res.data as VersionInitResult
        setVers((prev) =>
          prev.map((v) =>
            v.id === versionId
              ? { ...v, initialized: true, image: initResult.image_ref ?? v.image }
              : v
          )
        )
      } else {
        setInitErrors((prev) => ({ ...prev, [versionId]: res.error ?? '初始化失败' }))
      }
    } catch {
      setInitErrors((prev) => ({ ...prev, [versionId]: '网络错误' }))
    } finally {
      setInitializingId('')
    }
  }

  async function handleBuildKnowledge(versionId: string) {
    setBuildingId(versionId)
    try {
      const res = await versionsApi.buildKnowledge(versionId)
      if (res.ok && res.data) {
        // Optimistically update to "building" status
        setVers((prev) =>
          prev.map((v) =>
            v.id === versionId ? { ...v, kb_status: 'building' } : v
          )
        )
        // Poll for completion
        pollKBStatus(versionId)
      } else {
        setError(res.error ?? '启动知识库构建失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setBuildingId('')
    }
  }

  function pollKBStatus(versionId: string) {
    const interval = setInterval(async () => {
      const res = await versionsApi.get(versionId)
      if (res.ok && res.data) {
        const updated = res.data
        setVers((prev) =>
          prev.map((v) => v.id === versionId ? updated : v)
        )
        if (updated.kb_status === 'complete' || updated.kb_status === 'failed') {
          clearInterval(interval)
          // Also refresh language status (might have become active)
          if (id) {
            const langRes = await languages.get(id)
            if (langRes.ok && langRes.data) {
              setLang(langRes.data)
            }
          }
        }
      }
    }, 3000)
  }

  async function handleToggleKB(versionId: string) {
    if (expandedKB === versionId) {
      setExpandedKB(null)
      return
    }
    // If already loaded, just expand
    if (kbEntries[versionId]) {
      setExpandedKB(versionId)
      return
    }
    setLoadingKB(versionId)
    try {
      const res = await knowledge.list(versionId)
      if (res.ok && res.data) {
        setKBEntries((prev) => ({ ...prev, [versionId]: res.data! }))
        setExpandedKB(versionId)
      }
    } catch {
      // silently fail
    } finally {
      setLoadingKB(null)
    }
  }

  function kbStatusBadge(kbStatus: string) {
    switch (kbStatus) {
      case 'complete':
        return <Badge variant="success">知识库就绪</Badge>
      case 'building':
        return <Badge variant="info">构建中</Badge>
      case 'failed':
        return <Badge variant="danger">构建失败</Badge>
      default:
        return <Badge variant="warning">待构建</Badge>
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

      {/* Inactive language warning */}
      {lang.status !== 'active' && (
        <div className="mt-6 rounded-xl border border-amber-700/50 bg-amber-900/20 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="text-2xl">⚠️</span>
            <div>
              <p className="text-sm font-medium text-amber-200">
                知识库尚未构建 — 此语言当前不可用于学习
              </p>
              <p className="mt-1 text-xs text-amber-400/80">
                请先初始化版本环境，然后构建知识库。知识库完成后，语言将自动激活。
              </p>
            </div>
          </div>
        </div>
      )}

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
                <Badge variant={lang.compatibility_model === 'strict' ? 'info' : 'warning'}>
                  {lang.compatibility_model}
                </Badge>
                <span>·</span>
                <Badge variant={lang.status === 'active' ? 'success' : 'warning'}>
                  {lang.status === 'active' ? '已激活' : '未激活'}
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
            <p className="text-sm text-stone-400">版本发现未返回任何结果。</p>
            <p className="mt-2 text-sm text-stone-500">
              可能是该语言没有可访问的官网下载页，或网络原因导致抓取失败。可尝试重新添加。
            </p>
          </div>
        ) : (
          <div className="mt-4 divide-y divide-stone-800 rounded-xl border border-stone-800">
            {vers.map((v) => (
              <div key={v.id}>
                <div className="flex w-full items-center justify-between px-6 py-4">
                  <div className="flex items-center gap-3">
                    <span className="font-mono text-sm">{v.version}</span>
                    {v.initialized ? (
                      <Badge variant="success">就绪</Badge>
                    ) : (
                      <Badge variant="warning">未初始化</Badge>
                    )}
                    {v.status !== 'active' && (
                      <Badge variant="danger">已归档</Badge>
                    )}
                    {kbStatusBadge(v.kb_status)}
                    {v.image && (
                      <span className="text-xs text-stone-500 font-mono">{v.image}</span>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {/* Archive / Activate toggle */}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleToggleStatus(v.id, v.status)}
                    >
                      {v.status === 'active' ? '归档' : '激活'}
                    </Button>
                    {!v.initialized && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleInitialize(v.id)}
                        disabled={initializingId === v.id}
                      >
                        {initializingId === v.id ? '初始化中…' : '初始化环境'}
                      </Button>
                    )}
                    {v.initialized && (!v.kb_status || v.kb_status === 'pending' || v.kb_status === 'failed') && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleBuildKnowledge(v.id)}
                        disabled={buildingId === v.id}
                      >
                        {buildingId === v.id ? '启动中…' : v.kb_status === 'failed' ? '重试构建' : '构建知识库'}
                      </Button>
                    )}
                    {v.initialized && v.kb_status === 'building' && (
                      <span className="text-xs text-amber-400 animate-pulse">构建中…</span>
                    )}
                    {initErrors[v.id] && (
                      <span className="text-xs text-red-400">{initErrors[v.id]}</span>
                    )}
                    {v.initialized && v.kb_status === 'complete' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleKB(v.id)}
                      >
                        {expandedKB === v.id ? '收起知识库' : loadingKB === v.id ? '加载中…' : '查看知识库'}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => openPlayground(v.id)}
                      disabled={!v.initialized || lang.status !== 'active'}
                    >
                      {lang.status !== 'active' ? '未激活' : '运行'}
                    </Button>
                  </div>
                </div>

                {/* Inline playground panel */}
                {activeVersionId === v.id && (
                  <div className="border-t border-stone-800 bg-stone-900/50 px-6 py-4">
                    <div className="grid gap-4 lg:grid-cols-2">
                      {/* Code editor */}
                      <div>
                        <div className="flex items-center justify-between mb-3">
                          <span className="text-xs font-medium text-stone-500">
                            {lang.name} {v.version}
                          </span>
                          <div className="flex items-center gap-2">
                            <Button onClick={handleRun} disabled={running || !code.trim()} size="sm">
                              {running ? '运行中…' : '运行'}
                            </Button>
                            <button
                              onClick={() => setActiveVersionId('')}
                              className="text-xs text-stone-500 transition-colors hover:text-stone-300"
                            >
                              关闭
                            </button>
                          </div>
                        </div>
                        <textarea
                          value={code}
                          onChange={(e) => setCode(e.target.value)}
                          className="w-full h-48 rounded-lg border border-stone-700 bg-stone-950 px-3 py-2.5 font-mono text-sm text-stone-100 transition-colors focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500 resize-none"
                          placeholder="在此编写代码…"
                          spellCheck={false}
                        />
                      </div>

                      {/* Output */}
                      <div>
                        <span className="text-xs font-medium text-stone-500">输出</span>
                        <pre className="mt-3 h-48 overflow-auto rounded-lg border border-stone-800 bg-stone-950 px-3 py-2.5 font-mono text-sm text-stone-300 whitespace-pre-wrap break-words">
                          {output || '点击"运行"查看输出…'}
                        </pre>
                      </div>
                    </div>
                  </div>
                )}

                {/* Knowledge base section */}
                {expandedKB === v.id && kbEntries[v.id] && (
                  <div className="border-t border-stone-800 bg-stone-900/50 px-6 py-4">
                    <h4 className="text-xs font-medium text-stone-400 uppercase tracking-wide mb-3">
                      知识库 ({((kbEntries[v.id].shared ?? []).length + (kbEntries[v.id].private ?? []).length)} 条)
                    </h4>
                    <div className="space-y-3">
                      {['factual' as const, 'normative' as const].map((cat) => {
                        const sharedEntries = kbEntries[v.id].shared ?? []
                        const entries = sharedEntries.filter((e) => e.category === cat)
                        if (entries.length === 0) return null
                        const label = cat === 'factual' ? '事实性知识' : '规范性知识'
                        return (
                          <div key={cat}>
                            <p className="text-xs font-medium text-stone-500 mb-1.5">{label} ({entries.length})</p>
                            <div className="space-y-1.5">
                              {entries.map((entry) => {
                                const desc = (entry.content as Record<string, string>)?.description ?? ''
                                return (
                                  <details key={entry.id} className="group rounded-lg border border-stone-700/50 bg-stone-900 px-4 py-2.5">
                                    <summary className="cursor-pointer select-none text-sm font-medium text-stone-300 hover:text-stone-100">
                                      <span className="font-mono text-xs text-stone-500 mr-2">{entry.topic}</span>
                                      <span className="text-xs text-stone-400 truncate">{desc.slice(0, 80)}{desc.length > 80 ? '…' : ''}</span>
                                    </summary>
                                    <div className="mt-2 pl-2 border-l-2 border-amber-500/50">
                                      <pre className="text-xs text-stone-400 whitespace-pre-wrap leading-relaxed">
                                        {JSON.stringify(entry.content, null, 2)}
                                      </pre>
                                    </div>
                                  </details>
                                )
                              })}
                            </div>
                          </div>
                        )
                      })}
                      {kbEntries[v.id].shared.length === 0 && kbEntries[v.id].private.length === 0 && (
                        <p className="text-xs text-stone-500">暂无知识条目</p>
                      )}
                    </div>
                  </div>
                )}

                {/* Loading KB */}
                {loadingKB === v.id && (
                  <div className="border-t border-stone-800 bg-stone-900/50 px-6 py-4">
                    <p className="text-sm text-stone-500">加载知识库…</p>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}