import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import { languages, versions as versionsApi, execute, type Language, type LanguageVersion } from '../api/client'
import Card from '../components/ui/Card'
import Button from '../components/ui/Button'
import Skeleton from '../components/ui/Skeleton'

export default function PlaygroundPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()

  const { data: lang, loading: langLoading, error: langError } = useApi<Language>(
    () => languages.get(id!),
    { fetchOnMount: !!id },
  )
  const { data: vers, loading: versLoading } = useApi<LanguageVersion[]>(
    () => versionsApi.listByLanguage(id!),
    { fetchOnMount: !!id },
  )

  const [selectedVersionId, setSelectedVersionId] = useState(searchParams.get('version') ?? '')
  const [code, setCode] = useState('')
  const [output, setOutput] = useState('')
  const [running, setRunning] = useState(false)

  useEffect(() => {
    if (vers && vers.length > 0 && !selectedVersionId) {
      setSelectedVersionId(vers[0].id)
    }
  }, [vers, selectedVersionId])

  useEffect(() => {
    if (selectedVersionId) {
      setSearchParams({ version: selectedVersionId })
    }
  }, [selectedVersionId, setSearchParams])

  // Reset code when language changes
  useEffect(() => {
    setCode('')
  }, [lang])

  async function handleRun() {
    if (!selectedVersionId || !code.trim()) return
    setRunning(true)
    setOutput('')
    try {
      const res = await execute.run({
        version_id: selectedVersionId,
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

  if (langLoading || versLoading) {
    return (
      <div className="animate-fade-in">
        <Skeleton className="mb-1 h-7 w-48" />
        <Skeleton className="h-4 w-64" />
        <div className="mt-8 grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-96 rounded-xl" />
          <Skeleton className="h-96 rounded-xl" />
        </div>
      </div>
    )
  }

  if (langError || !lang) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-sm text-danger">
          {langError || '语言未找到'}
        </p>
      </div>
    )
  }

  return (
    <div className="animate-fade-in">
      <h1 className="text-2xl font-bold">Playground</h1>
      <p className="mt-1 text-sm text-text-secondary">
        {lang.name} — 在线编写和运行代码
      </p>

      <div className="mt-8 grid gap-6 lg:grid-cols-2">
        {/* Editor */}
        <Card className="p-5">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h2 className="text-sm font-semibold text-text-primary">代码</h2>
              <select
                value={selectedVersionId}
                onChange={(e) => setSelectedVersionId(e.target.value)}
                className="rounded-md border border-border bg-bg-subtle px-2.5 py-1 text-xs text-text-primary transition-colors focus:border-accent focus:outline-none"
              >
                {(!vers || vers.length === 0) && (
                  <option value="">无可用版本</option>
                )}
                {vers?.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.version}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={handleRun}
              disabled={
                running ||
                !vers ||
                vers.length === 0 ||
                !code.trim()
              }
              size="sm"
            >
              {running ? '运行中…' : '运行'}
            </Button>
          </div>

          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="h-64 w-full resize-none rounded-lg border border-border bg-bg-base px-4 py-3 font-mono text-sm text-text-primary placeholder-text-muted transition-colors focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="在此编写代码…"
            spellCheck={false}
          />
        </Card>

        {/* Output */}
        <Card className="p-5">
          <h2 className="mb-4 text-sm font-semibold text-text-primary">输出</h2>
          <pre className="h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border bg-bg-base px-4 py-3 font-mono text-sm text-text-secondary">
            {output || '点击"运行"查看输出…'}
          </pre>
        </Card>
      </div>

      {(!vers || vers.length === 0) && (
        <div className="mt-6 rounded-xl border border-border bg-bg-elevated px-6 py-6 text-center">
          <p className="text-sm text-text-secondary">
            此语言尚无可用的运行时版本。
          </p>
          <p className="mt-1 text-sm text-text-muted">
            请先在语言详情页添加版本。
          </p>
        </div>
      )}
    </div>
  )
}
