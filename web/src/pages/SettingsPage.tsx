import { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'
import { config, type LLMConfig, type LLMProvider } from '../api/client'
import Card from '../components/ui/Card'
import Button from '../components/ui/Button'
import Skeleton from '../components/ui/Skeleton'

const emptyProvider = (): LLMProvider => ({
  name: '',
  endpoint: '',
  models: [''],
  api_key: '',
})

export default function SettingsPage() {
  const { data: loadedCfg, loading, error: loadError, refetch } = useApi<LLMConfig>(
    () => config.getLLM(),
    { fetchOnMount: true },
  )
  const [cfg, setCfg] = useState<LLMConfig | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  // Sync local state when loaded
  useEffect(() => {
    if (loadedCfg) setCfg(loadedCfg)
  }, [loadedCfg])

  function updateProvider(index: number, patch: Partial<LLMProvider>) {
    if (!cfg) return
    setCfg({
      ...cfg,
      providers: cfg.providers.map((p, i) =>
        i === index ? { ...p, ...patch } : p,
      ),
    })
  }

  function updateModel(providerIdx: number, modelIdx: number, value: string) {
    if (!cfg) return
    setCfg({
      ...cfg,
      providers: cfg.providers.map((p, i) => {
        if (i !== providerIdx) return p
        return { ...p, models: p.models.map((m, j) => (j === modelIdx ? value : m)) }
      }),
    })
  }

  function addModel(providerIdx: number) {
    if (!cfg) return
    setCfg({
      ...cfg,
      providers: cfg.providers.map((p, i) =>
        i === providerIdx ? { ...p, models: [...p.models, ''] } : p,
      ),
    })
  }

  function removeModel(providerIdx: number, modelIdx: number) {
    if (!cfg) return
    setCfg({
      ...cfg,
      providers: cfg.providers.map((p, i) => {
        if (i !== providerIdx) return p
        return { ...p, models: p.models.filter((_, j) => j !== modelIdx) }
      }),
    })
  }

  function addProvider() {
    if (!cfg) return
    setCfg({ ...cfg, providers: [...cfg.providers, emptyProvider()] })
  }

  function removeProvider(index: number) {
    if (!cfg) return
    setCfg({
      ...cfg,
      providers: cfg.providers.filter((_, i) => i !== index),
    })
  }

  async function handleSave() {
    if (!cfg) return
    const previousCfg = { ...cfg }
    setSaving(true)
    setError('')
    setSaved(false)

    // Optimistic update: mark saved immediately
    setSaved(true)

    try {
      const res = await config.updateLLM(cfg)
      if (res.ok && res.data) {
        setCfg(res.data)
      } else {
        // Rollback
        setCfg(previousCfg)
        setError(res.error ?? '保存失败')
        setSaved(false)
      }
    } catch {
      setCfg(previousCfg)
      setError('网络错误')
      setSaved(false)
    } finally {
      setSaving(false)
    }
  }

  const inputClass =
    'rounded-md border border-border bg-bg-subtle px-3 py-2 text-sm text-text-primary placeholder-text-muted transition-colors focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent'
  const labelClass = 'text-xs font-medium text-text-muted'

  if (loading) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 className="text-2xl font-bold">LLM 设置</h1>
        <div className="mt-8 space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 className="text-2xl font-bold">LLM 设置</h1>
        <p className="mt-8 text-center text-sm text-danger">{loadError}</p>
        <div className="mt-4 flex justify-center">
          <Button onClick={refetch} variant="secondary">重试</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="animate-fade-in text-2xl font-bold">LLM 设置</h1>
      <p className="mt-2 text-sm text-text-secondary">
        配置 LLM 提供商用于代码生成和分析。
      </p>

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}
      {saved && !error && (
        <p className="mt-4 animate-fade-in text-sm text-success">配置已保存。</p>
      )}

      {cfg && (
        <div className="mt-8 space-y-8">
          {/* Default provider */}
          <div>
            <label className="flex flex-col gap-1.5">
              <span className={labelClass}>默认提供商</span>
              <select
                value={cfg.default}
                onChange={(e) => setCfg({ ...cfg, default: e.target.value })}
                className={`${inputClass} appearance-none bg-bg-subtle bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2216%22%20height%3D%2216%22%20fill%3D%22none%22%20stroke%3D%22%23a89090%22%20stroke-width%3D%222%22%3E%3Cpath%20d%3D%22m4%206%204%204%204-4%22%2F%3E%3C%2Fsvg%3E')] bg-[length:16px_16px] bg-[right_12px_center] bg-no-repeat pr-10`}
              >
                {cfg.providers.map((p) => (
                  <option key={p.name} value={p.name}>
                    {p.name || '(未命名)'}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {/* Providers */}
          <div>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-text-primary">提供商</h2>
              <Button variant="ghost" size="sm" onClick={addProvider}>
                + 添加提供商
              </Button>
            </div>

            <div className="mt-4 space-y-4">
              {cfg.providers.map((p, i) => (
                <Card key={i} className="p-4">
                  <div className="mb-4 flex items-center justify-between">
                    <span className="text-xs font-medium text-text-muted">
                      提供商 {i + 1}
                    </span>
                    <button
                      onClick={() => removeProvider(i)}
                      className="text-xs text-danger transition-colors hover:text-danger"
                    >
                      移除
                    </button>
                  </div>

                  <div className="space-y-3">
                    <label className="flex flex-col gap-1.5">
                      <span className={labelClass}>名称</span>
                      <input
                        type="text"
                        value={p.name}
                        onChange={(e) => updateProvider(i, { name: e.target.value })}
                        placeholder="例如: DeepSeek"
                        className={inputClass}
                      />
                    </label>

                    <div>
                      <div className="mb-1.5 flex items-center justify-between">
                        <span className={labelClass}>模型列表</span>
                        <button
                          onClick={() => addModel(i)}
                          className="text-xs text-accent transition-colors hover:text-accent-hover"
                        >
                          + 添加模型
                        </button>
                      </div>
                      <div className="space-y-1.5">
                        {p.models.map((m, mi) => (
                          <div key={mi} className="flex items-center gap-2">
                            <input
                              type="text"
                              value={m}
                              onChange={(e) => updateModel(i, mi, e.target.value)}
                              placeholder="例如: deepseek-chat"
                              className={`flex-1 font-mono ${inputClass}`}
                            />
                            {p.models.length > 1 && (
                              <button
                                onClick={() => removeModel(i, mi)}
                                className="text-xs text-danger transition-colors hover:text-danger"
                              >
                                ✕
                              </button>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>

                    <label className="flex flex-col gap-1.5">
                      <span className={labelClass}>Endpoint</span>
                      <input
                        type="text"
                        value={p.endpoint}
                        onChange={(e) => updateProvider(i, { endpoint: e.target.value })}
                        placeholder="https://api.deepseek.com/v1"
                        className={`font-mono ${inputClass}`}
                      />
                      <p className="text-xs text-accent">
                        容器内请用 host.docker.internal 代替 localhost
                      </p>
                    </label>

                    <label className="flex flex-col gap-1.5">
                      <span className={labelClass}>API Key</span>
                      <input
                        type="password"
                        value={p.api_key}
                        onChange={(e) => updateProvider(i, { api_key: e.target.value })}
                        placeholder="留空以保留现有密钥"
                        autoComplete="off"
                        className={`font-mono ${inputClass}`}
                      />
                    </label>
                  </div>
                </Card>
              ))}
            </div>

            {cfg.providers.length === 0 && (
              <p className="mt-4 text-sm text-text-muted">
                尚未配置提供商。请至少添加一个。
              </p>
            )}
          </div>

          <Button
            onClick={handleSave}
            disabled={saving || cfg.providers.length === 0}
            className="w-full"
          >
            {saving ? '保存中…' : '保存配置'}
          </Button>
        </div>
      )}
    </div>
  )
}
