import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { languages, type InitSuggestion } from '../api/client'
import { isIconURL, friendlyError } from '../lib/utils'
import { useStream } from '../hooks/useStream'
import Card from '../components/ui/Card'
import Badge from '../components/ui/Badge'
import Button from '../components/ui/Button'

interface ProgressStep {
  step: string
  status: string
  message: string
}

const STEP_LABELS: Record<string, string> = {
  wikipedia_search: '搜索 Wikipedia',
  wikipedia_categories: '验证语言分类',
  wikipedia_infobox: '解析信息框',
  llm_analyze: 'LLM 分析语言',
  sync_query: '同步查询',
}

const STEP_ORDER = [
  'wikipedia_search',
  'wikipedia_categories',
  'wikipedia_infobox',
  'llm_analyze',
  'sync_query',
]

export default function AddLanguagePage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [querying, setQuerying] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState('')
  const [suggestion, setSuggestion] = useState<InitSuggestion | null>(null)
  const [stepMap, setStepMap] = useState<Record<string, ProgressStep>>({})
  const streamProgressRef = useRef(false)
  const streamResultRef = useRef(false)

  const { start: startStream, cancel: cancelStream } = useStream({
    onLine: (line) => {
      if ('ok' in line && line.ok) {
        streamResultRef.current = true
        setSuggestion(line.data as InitSuggestion)
        setQuerying(false)
        return
      }
      streamProgressRef.current = true
      const step = line as unknown as ProgressStep
      setStepMap((prev) => ({ ...prev, [step.step]: step }))
      if (step.status === 'error' && step.step === 'fatal') {
        setError(friendlyError(step.message))
        setQuerying(false)
      }
    },
    onDone: () => setQuerying(false),
    onError: (msg) => {
      setError(friendlyError(msg))
      setQuerying(false)
    },
  })

  async function handleQuery() {
    if (!name.trim()) return
    setQuerying(true)
    setError('')
    setSuggestion(null)
    setStepMap({})
    streamProgressRef.current = false
    streamResultRef.current = false

    const result = await startStream((signal) =>
      languages.initQueryStream(name.trim(), signal),
    )

    // Fallback to non-streaming on transport failure (no progress received)
    if (result === 'done' && !streamResultRef.current && !streamProgressRef.current) {
      setError('流式查询不可用，请重试')
    }
  }

  function handleCancelQuery() {
    cancelStream()
    setQuerying(false)
    setStepMap({})
    setError('')
  }

  function handleCancel() {
    setSuggestion(null)
    setStepMap({})
  }

  async function handleConfirm() {
    if (!suggestion) return
    setConfirming(true)
    setError('')
    try {
      const allVersions = suggestion.versions?.map((v) => v.version) ?? []
      const res = await languages.initConfirm(
        {
          name: suggestion.name,
          slug: suggestion.slug,
          wiki_title: suggestion.wiki_title ?? "",
          icon: suggestion.icon,
          compatibility_model: suggestion.compatibility_model,
          docs_url: '',
          runtime_url: '',
          versions: allVersions,
          discovered_versions: suggestion.versions,
        },
      )
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

  function handleCancelConfirm() {
    setConfirming(false)
  }

  const visibleSteps = STEP_ORDER.filter(
    (key) => key in stepMap && stepMap[key].step !== 'fatal',
  ).map((key) => stepMap[key])

  const showProgress = visibleSteps.length > 0 && !suggestion

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="text-2xl font-bold">添加语言</h1>
      <p className="mt-2 text-sm text-text-secondary">
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
          className="flex-1 rounded-lg border border-border bg-bg-elevated px-4 py-2.5 text-sm text-text-primary placeholder-text-muted transition-colors focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
        />
        <Button onClick={handleQuery} disabled={querying || !name.trim()}>
          {querying ? '查询中…' : '查找语言'}
        </Button>
      </div>

      {/* Progress steps */}
      {showProgress && (
        <div className="mt-6 animate-slide-down rounded-lg border border-border bg-bg-elevated px-5 py-4">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs font-medium tracking-wide uppercase text-text-muted">
              查询进度
            </p>
            {querying && (
              <Button onClick={handleCancelQuery} variant="ghost" size="sm">
                取消查询
              </Button>
            )}
          </div>
          <div className="space-y-2">
            {visibleSteps.map((s) => {
              const label = STEP_LABELS[s.step] ?? s.step
              const isRunning = s.status === 'running'
              const isDone = s.status === 'done'
              const isError = s.status === 'error'
              return (
                <div key={s.step} className="flex items-center gap-3 text-sm">
                  <span
                    className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[10px] ${
                      isDone
                        ? 'bg-success-bg text-success'
                        : isError
                          ? 'bg-danger-bg text-danger'
                          : 'bg-accent-faint text-accent'
                    }`}
                  >
                    {isDone ? '✓' : isError ? '✗' : '●'}
                  </span>
                  <span
                    className={
                      isRunning
                        ? 'text-accent'
                        : isError
                          ? 'text-danger'
                          : 'text-text-secondary'
                    }
                  >
                    {label}
                  </span>
                  {isRunning && (
                    <span className="animate-pulse text-xs text-text-muted">
                      {s.message}
                    </span>
                  )}
                  {isError && (
                    <span className="text-xs text-danger">{s.message}</span>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      {/* Read-only confirmation card */}
      {suggestion && (
        <Card className="mt-8 animate-fade-in-up overflow-hidden">
          {/* Header */}
          <div className="flex items-center gap-4 bg-bg-hover/50 px-6 py-5">
            {suggestion.icon && isIconURL(suggestion.icon) ? (
              <img
                src={suggestion.icon}
                alt=""
                width={48}
                height={48}
                className="h-12 w-12 shrink-0 rounded-lg object-contain"
              />
            ) : (
              <span className="text-4xl" aria-hidden="true">
                {suggestion.icon || '📄'}
              </span>
            )}
            <div className="min-w-0">
              <h2 className="text-xl font-bold">{suggestion.name}</h2>
              <div className="mt-1.5 flex items-center gap-2 text-sm text-text-secondary">
                <code className="rounded bg-bg-subtle px-1.5 py-0.5 font-mono text-xs">
                  {suggestion.slug}
                </code>
                <span>·</span>
                <Badge
                  variant={
                    suggestion.compatibility_model === 'strict'
                      ? 'info'
                      : suggestion.compatibility_model === 'versioned'
                        ? 'warning'
                        : 'default'
                  }
                >
                  {suggestion.compatibility_model}
                </Badge>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="space-y-4 px-6 py-5">
            <p className="text-sm leading-relaxed text-text-primary">
              {suggestion.description}
            </p>

            {/* Versions preview */}
            {suggestion.versions && suggestion.versions.length > 0 && (
              <div>
                <p className="mb-2 text-xs font-medium tracking-wide uppercase text-text-muted">
                  收录版本 ({suggestion.versions.length})
                </p>
                <div className="space-y-2">
                  {suggestion.versions.map((v) => (
                    <div
                      key={v.version}
                      className="flex items-center gap-3 rounded-lg border border-border bg-bg-base px-4 py-3"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm font-medium">
                            {v.version}
                          </span>
                          {v.lts && <Badge variant="success">LTS</Badge>}
                          {v.version === suggestion.latest_version && (
                            <Badge variant="info">最新</Badge>
                          )}
                        </div>
                        <p className="mt-0.5 truncate text-xs text-text-muted">
                          {v.brief}
                        </p>
                      </div>
                      <span className="shrink-0 text-xs text-text-muted">
                        {v.released}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex gap-3 border-t border-border px-6 py-4">
            {confirming ? (
              <Button onClick={handleCancelConfirm} variant="danger" className="flex-1">
                取消创建
              </Button>
            ) : (
              <>
                <Button onClick={handleConfirm} className="flex-1">
                  确认添加
                </Button>
                <Button onClick={handleCancel} variant="secondary" className="flex-1">
                  取消
                </Button>
              </>
            )}
          </div>
        </Card>
      )}
    </div>
  )
}
