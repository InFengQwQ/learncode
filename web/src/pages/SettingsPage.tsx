import { useState, useEffect } from 'react'
import { config, type LLMConfig, type LLMProvider } from '../api/client'
import Card from '../components/ui/Card'
import Button from '../components/ui/Button'

const emptyProvider = (): LLMProvider => ({
  name: '',
  endpoint: '',
  models: [''],
  api_key: '',
})

export default function SettingsPage() {
  const [cfg, setCfg] = useState<LLMConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  async function loadConfig() {
    setLoading(true)
    setError('')
    try {
      const res = await config.getLLM()
      if (res.ok && res.data) {
        setCfg(res.data)
      } else {
        setError(res.error ?? '加载配置失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfig()
  }, [])

  function updateProvider(index: number, patch: Partial<LLMProvider>) {
    if (!cfg) return
    const providers = cfg.providers.map((p, i) =>
      i === index ? { ...p, ...patch } : p,
    )
    setCfg({ ...cfg, providers })
  }

  function updateModel(providerIdx: number, modelIdx: number, value: string) {
    if (!cfg) return
    const providers = cfg.providers.map((p, i) => {
      if (i !== providerIdx) return p
      const models = p.models.map((m, j) => (j === modelIdx ? value : m))
      return { ...p, models }
    })
    setCfg({ ...cfg, providers })
  }

  function addModel(providerIdx: number) {
    if (!cfg) return
    const providers = cfg.providers.map((p, i) => {
      if (i !== providerIdx) return p
      return { ...p, models: [...p.models, ''] }
    })
    setCfg({ ...cfg, providers })
  }

  function removeModel(providerIdx: number, modelIdx: number) {
    if (!cfg) return
    const providers = cfg.providers.map((p, i) => {
      if (i !== providerIdx) return p
      return { ...p, models: p.models.filter((_, j) => j !== modelIdx) }
    })
    setCfg({ ...cfg, providers })
  }

  function addProvider() {
    if (!cfg) return
    setCfg({ ...cfg, providers: [...cfg.providers, emptyProvider()] })
  }

  function removeProvider(index: number) {
    if (!cfg) return
    setCfg({ ...cfg, providers: cfg.providers.filter((_, i) => i !== index) })
  }

  async function handleSave() {
    if (!cfg) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const res = await config.updateLLM(cfg)
      if (res.ok && res.data) {
        setCfg(res.data)
        setSaved(true)
      } else {
        setError(res.error ?? '保存失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setSaving(false)
    }
  }

  const inputClass =
    'rounded-md border border-stone-700 bg-stone-800 px-3 py-2 text-sm text-stone-100 placeholder-stone-500 transition-colors focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500'
  const labelClass = 'text-xs font-medium text-stone-500'

  if (loading) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 className="text-2xl font-bold">LLM 设置</h1>
        <p className="mt-8 text-center text-sm text-stone-500">加载中…</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-bold">LLM 设置</h1>
      <p className="mt-2 text-sm text-stone-400">
        配置 LLM 提供商用于代码生成和分析。
      </p>

      {error && <p className="mt-4 text-sm text-red-400">{error}</p>}
      {saved && (
        <p className="mt-4 text-sm text-emerald-400">配置已保存。</p>
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
                className={`${inputClass} appearance-none bg-stone-800 bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2216%22%20height%3D%2216%22%20fill%3D%22none%22%20stroke%3D%22%23a8a29e%22%20stroke-width%3D%222%22%3E%3Cpath%20d%3D%22m4%206%204%204%204-4%22%2F%3E%3C%2Fsvg%3E')] bg-[length:16px_16px] bg-[right_12px_center] bg-no-repeat pr-10`}
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
              <h2 className="text-sm font-semibold text-stone-300">提供商</h2>
              <Button variant="ghost" size="sm" onClick={addProvider}>
                + 添加提供商
              </Button>
            </div>

            <div className="mt-4 space-y-4">
              {cfg.providers.map((p, i) => (
                <Card key={i} className="p-4">
                  <div className="flex items-center justify-between mb-4">
                    <span className="text-xs font-medium text-stone-500">
                      提供商 {i + 1}
                    </span>
                    <button
                      onClick={() => removeProvider(i)}
                      className="text-xs text-red-400 transition-colors hover:text-red-300"
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
                        onChange={(e) =>
                          updateProvider(i, { name: e.target.value })
                        }
                        placeholder="例如: DeepSeek"
                        className={inputClass}
                      />
                    </label>

                    <div>
                      <div className="flex items-center justify-between mb-1.5">
                        <span className={labelClass}>模型列表</span>
                        <button
                          onClick={() => addModel(i)}
                          className="text-xs text-amber-500 transition-colors hover:text-amber-400"
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
                              onChange={(e) =>
                                updateModel(i, mi, e.target.value)
                              }
                              placeholder="例如: deepseek-chat"
                              className={`flex-1 font-mono ${inputClass}`}
                            />
                            {p.models.length > 1 && (
                              <button
                                onClick={() => removeModel(i, mi)}
                                className="text-xs text-red-400 transition-colors hover:text-red-300"
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
                        onChange={(e) =>
                          updateProvider(i, { endpoint: e.target.value })
                        }
                        placeholder="https://api.deepseek.com/v1"
                        className={`font-mono ${inputClass}`}
                      />
                    </label>

                    <label className="flex flex-col gap-1.5">
                      <span className={labelClass}>API Key</span>
                      <input
                        type="password"
                        value={p.api_key}
                        onChange={(e) =>
                          updateProvider(i, { api_key: e.target.value })
                        }
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
              <p className="mt-4 text-sm text-stone-500">
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
