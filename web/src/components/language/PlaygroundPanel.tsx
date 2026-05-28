import { useState } from 'react'
import { execute } from '../../api/client'
import Button from '../ui/Button'

interface PlaygroundPanelProps {
  versionId: string
  languageName: string
  versionLabel: string
  languageActive: boolean
  onClose: () => void
}

export default function PlaygroundPanel({
  versionId,
  languageName,
  versionLabel,
  languageActive,
  onClose,
}: PlaygroundPanelProps) {
  const [code, setCode] = useState('')
  const [output, setOutput] = useState('')
  const [running, setRunning] = useState(false)

  async function handleRun() {
    if (!code.trim()) return
    setRunning(true)
    setOutput('')
    try {
      const res = await execute.run({ version_id: versionId, code: code.trim() })
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

  return (
    <div className="border-t border-border bg-bg-base/50 px-6 py-4">
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Code editor */}
        <div>
          <div className="mb-3 flex items-center justify-between">
            <span className="text-xs font-medium text-text-muted">
              {languageName} {versionLabel}
            </span>
            <div className="flex items-center gap-2">
              <Button onClick={handleRun} disabled={!code.trim() || running || !languageActive} size="sm">
                {running ? '运行中…' : '运行'}
              </Button>
              <button
                onClick={onClose}
                className="text-xs text-text-muted transition-colors hover:text-text-primary"
              >
                关闭
              </button>
            </div>
          </div>
          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="h-48 w-full resize-none rounded-lg border border-border bg-bg-base px-3 py-2.5 font-mono text-sm text-text-primary transition-colors focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder="在此编写代码…"
            aria-label="代码编辑器"
            spellCheck={false}
          />
        </div>

        {/* Output */}
        <div>
          <span className="text-xs font-medium text-text-muted">输出</span>
          <pre className="mt-3 h-48 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border bg-bg-base px-3 py-2.5 font-mono text-sm text-text-secondary">
            {output || '点击"运行"查看输出…'}
          </pre>
        </div>
      </div>
    </div>
  )
}
