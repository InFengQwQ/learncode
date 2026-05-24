import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { languages, versions, execute, type Language, type LanguageVersion } from '../api/client'
import Card from '../components/ui/Card'
import Button from '../components/ui/Button'

export default function PlaygroundPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()

  const [lang, setLang] = useState<Language | null>(null)
  const [vers, setVers] = useState<LanguageVersion[]>([])
  const [selectedVersionId, setSelectedVersionId] = useState(searchParams.get('version') ?? '')
  const [code, setCode] = useState('')
  const [output, setOutput] = useState('')
  const [running, setRunning] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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
          if (verRes.data.length > 0 && !selectedVersionId) {
            setSelectedVersionId(verRes.data[0].id)
          }
        }
      } else {
        setError(langRes.error ?? '加载失败')
      }
      setLoading(false)
    })
  }, [id])

  useEffect(() => {
    if (selectedVersionId) {
      setSearchParams({ version: selectedVersionId })
    }
  }, [selectedVersionId])

  // Set default code snippet based on language
  useEffect(() => {
    if (!lang) return
    switch (lang.slug) {
      case 'python':
        setCode('print("Hello, World!")')
        break
      case 'javascript':
      case 'node':
      case 'nodejs':
        setCode('console.log("Hello, World!")')
        break
      default:
        setCode('// Write your code here')
    }
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
      <h1 className="text-2xl font-bold">Playground</h1>
      <p className="mt-1 text-sm text-stone-400">
        {lang.name} — 在线编写和运行代码
      </p>

      <div className="mt-8 grid gap-6 lg:grid-cols-2">
        {/* Editor */}
        <Card className="p-5">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <h2 className="text-sm font-semibold text-stone-300">代码</h2>
              <select
                value={selectedVersionId}
                onChange={(e) => setSelectedVersionId(e.target.value)}
                className="rounded-md border border-stone-700 bg-stone-800 px-2.5 py-1 text-xs text-stone-300 transition-colors focus:border-amber-500 focus:outline-none"
              >
                {vers.length === 0 && <option value="">无可用版本</option>}
                {vers.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.version}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={handleRun}
              disabled={running || vers.length === 0 || !code.trim()}
              size="sm"
            >
              {running ? '运行中…' : '运行'}
            </Button>
          </div>

          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="w-full h-64 rounded-lg border border-stone-700 bg-stone-950 px-4 py-3 font-mono text-sm text-stone-100 placeholder-stone-500 transition-colors focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500 resize-none"
            placeholder="在此编写代码…"
            spellCheck={false}
          />
        </Card>

        {/* Output */}
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-stone-300 mb-4">输出</h2>
          <pre className="h-64 overflow-auto rounded-lg border border-stone-800 bg-stone-950 px-4 py-3 font-mono text-sm text-stone-300 whitespace-pre-wrap break-words">
            {output || '点击"运行"查看输出…'}
          </pre>
        </Card>
      </div>

      {vers.length === 0 && (
        <div className="mt-6 rounded-xl border border-stone-800 bg-stone-900 px-6 py-6 text-center">
          <p className="text-sm text-stone-400">
            此语言尚无可用的运行时版本。
          </p>
          <p className="mt-1 text-sm text-stone-500">
            请先在语言详情页添加版本。
          </p>
        </div>
      )}
    </div>
  )
}
