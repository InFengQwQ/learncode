import type { KnowledgeEntry } from '../../api/client'

interface KnowledgeBasePanelProps {
  entries: {
    shared: KnowledgeEntry[]
    private: KnowledgeEntry[]
  }
}

export default function KnowledgeBasePanel({ entries }: KnowledgeBasePanelProps) {
  const totalCount = (entries.shared?.length ?? 0) + (entries.private?.length ?? 0)
  const categories = ['factual' as const, 'normative' as const]
  const categoryLabels = { factual: '事实性知识', normative: '规范性知识' }

  return (
    <div className="border-t border-border bg-bg-base/50 px-6 py-4">
      <h4 className="mb-3 text-xs font-medium tracking-wide uppercase text-text-muted">
        知识库 ({totalCount} 条)
      </h4>
      <div className="space-y-3">
        {categories.map((cat) => {
          const sharedEntries = entries.shared ?? []
          const filtered = sharedEntries.filter((e) => e.category === cat)
          if (filtered.length === 0) return null
          return (
            <div key={cat}>
              <p className="mb-1.5 text-xs font-medium text-text-muted">
                {categoryLabels[cat]} ({filtered.length})
              </p>
              <div className="space-y-1.5">
                {filtered.map((entry) => {
                  const desc =
                    (entry.content as Record<string, string>)?.description ?? ''
                  return (
                    <details
                      key={entry.id}
                      className="group rounded-lg border border-border/50 bg-bg-base px-4 py-2.5"
                    >
                      <summary className="cursor-pointer select-none text-sm font-medium text-text-secondary hover:text-text-primary">
                        <span className="mr-2 font-mono text-xs text-text-muted">
                          {entry.topic}
                        </span>
                        <span className="text-xs text-text-muted">
                          {desc.slice(0, 80)}
                          {desc.length > 80 ? '…' : ''}
                        </span>
                      </summary>
                      <div className="mt-2 border-l-2 border-accent/50 pl-2">
                        <pre className="whitespace-pre-wrap text-xs leading-relaxed text-text-muted">
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
        {entries.shared.length === 0 && entries.private.length === 0 && (
          <p className="text-xs text-text-muted">暂无知识条目</p>
        )}
      </div>
    </div>
  )
}
