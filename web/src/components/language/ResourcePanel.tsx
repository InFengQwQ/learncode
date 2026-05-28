import type { ResearchResult } from '../../api/client'
import Card from '../ui/Card'
import Badge from '../ui/Badge'

function ResourceSection({
  title,
  items,
}: {
  title: string
  items: ResearchResult['docs']
}) {
  const list = items ?? []
  return (
    <Card>
      <div className="px-5 py-4">
        <h3 className="text-sm font-medium text-text-primary">
          {title} ({list.length})
        </h3>
        {list.length === 0 ? (
          <p className="mt-2 text-xs text-text-muted">未发现{title}资源</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {list.map((item, i) => (
              <li key={i}>
                <a
                  href={item.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block rounded-lg border border-border bg-bg-base px-3 py-2 transition-colors hover:border-accent/50 hover:bg-bg-hover"
                >
                  <span className="text-xs font-medium text-text-primary">
                    {item.description}
                  </span>
                  <div className="mt-1 flex items-center gap-2">
                    <Badge
                      variant={item.authority === 'official' ? 'success' : 'info'}
                    >
                      {item.authority}
                    </Badge>
                    <span className="truncate text-xs text-text-muted">
                      {item.url}
                    </span>
                  </div>
                </a>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}

interface ResourcePanelProps {
  result: ResearchResult
}

export default function ResourcePanel({ result }: ResourcePanelProps) {
  return (
    <div className="mt-6">
      <h2 className="mb-4 text-lg font-semibold">发现的资源</h2>
      <div className="grid gap-4 lg:grid-cols-3">
        <ResourceSection title="文档" items={result.docs ?? []} />
        <ResourceSection title="运行时" items={result.runtimes ?? []} />
        <ResourceSection title="规范标准" items={result.specs ?? []} />
      </div>
    </div>
  )
}
