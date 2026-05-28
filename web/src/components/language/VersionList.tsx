import { useState, useRef } from 'react'
import type { Language, LanguageVersion } from '../../api/client'
import { languages, versions as versionsApi, knowledge } from '../../api/client'
import Badge from '../ui/Badge'
import Button from '../ui/Button'
import KnowledgeStatusBadge from './KnowledgeStatusBadge'
import VersionActions from './VersionActions'
import PlaygroundPanel from './PlaygroundPanel'
import KnowledgeBasePanel from './KnowledgeBasePanel'

interface VersionListProps {
  versions: LanguageVersion[]
  language: Language
  onVersionsChange: (versions: LanguageVersion[]) => void
}

export default function VersionList({
  versions: vers,
  language: lang,
  onVersionsChange,
}: VersionListProps) {
  const [initializingId, setInitializingId] = useState('')
  const [initErrors, setInitErrors] = useState<Record<string, string>>({})
  const [buildingId, setBuildingId] = useState('')
  const [expandedKB, setExpandedKB] = useState<string | null>(null)
  const [kbEntries, setKBEntries] = useState<
    Record<string, { shared: import('../../api/client').KnowledgeEntry[]; private: import('../../api/client').KnowledgeEntry[] }>
  >({})
  const [loadingKB, setLoadingKB] = useState<string | null>(null)
  const [activeVersionId, setActiveVersionId] = useState('')
  const [discoveringVersions, setDiscoveringVersions] = useState(false)
  const [discoverError, setDiscoverError] = useState('')
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  async function handleToggleStatus(versionId: string, currentStatus: string) {
    const newStatus = currentStatus === 'active' ? 'archived' : 'active'
    try {
      const res = await versionsApi.setStatus(versionId, newStatus)
      if (res.ok && res.data) {
        onVersionsChange(vers.map((v) => (v.id === versionId ? res.data! : v)))
      }
    } catch {
      // silently fail
    }
  }

  async function handleInitialize(versionId: string) {
    setInitializingId(versionId)
    setInitErrors((prev) => {
      const next = { ...prev }
      delete next[versionId]
      return next
    })
    try {
      const res = await versionsApi.initialize(versionId)
      if (res.ok && res.data) {
        onVersionsChange(
          vers.map((v) =>
            v.id === versionId
              ? { ...v, initialized: true, image: res.data!.image_ref ?? v.image }
              : v,
          ),
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

  function handleCancelInit() {
    setInitializingId('')
  }

  async function handleBuildKnowledge(versionId: string) {
    setBuildingId(versionId)
    try {
      const res = await versionsApi.buildKnowledge(versionId)
      if (res.ok && res.data) {
        onVersionsChange(
          vers.map((v) =>
            v.id === versionId ? { ...v, kb_status: 'building' } : v,
          ),
        )
        pollKBStatus(versionId)
      } else {
        // Error is handled silently
      }
    } catch {
      // silently fail
    } finally {
      setBuildingId('')
    }
  }

  function pollKBStatus(versionId: string) {
    if (pollIntervalRef.current) clearInterval(pollIntervalRef.current)
    const interval = setInterval(async () => {
      const res = await versionsApi.get(versionId)
      if (res.ok && res.data) {
        const updated = res.data
        onVersionsChange(vers.map((v) => (v.id === versionId ? updated : v)))
        if (updated.kb_status === 'complete' || updated.kb_status === 'failed') {
          clearInterval(interval)
          pollIntervalRef.current = null
        }
      }
    }, 3000)
    pollIntervalRef.current = interval
  }

  function handleStopPolling(versionId: string) {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current)
      pollIntervalRef.current = null
    }
    onVersionsChange(
      vers.map((v) =>
        v.id === versionId ? { ...v, kb_status: 'pending' as const } : v,
      ),
    )
  }

  async function handleToggleKB(versionId: string) {
    if (expandedKB === versionId) {
      setExpandedKB(null)
      return
    }
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

  async function handleDiscoverVersions() {
    setDiscoveringVersions(true)
    setDiscoverError('')
    try {
      const res = await languages.discoverVersions(lang.id)
      if (res.ok) {
        const verRes = await versionsApi.listByLanguage(lang.id)
        if (verRes.ok && verRes.data) {
          onVersionsChange(verRes.data)
        }
      } else {
        setDiscoverError(res.error ?? '历史版本发现失败')
      }
    } catch {
      setDiscoverError('网络错误')
    } finally {
      setDiscoveringVersions(false)
    }
  }

  return (
    <div className="mt-10">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">语言版本</h2>
        {lang.compatibility_model === 'versioned' && (
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleDiscoverVersions}
              disabled={discoveringVersions}
            >
              {discoveringVersions ? '发现中…' : '发现历史版本'}
            </Button>
          </div>
        )}
      </div>

      {discoverError && (
        <div className="mt-3 rounded-lg border border-danger-bg bg-danger-bg/50 px-4 py-2">
          <p className="text-sm text-danger">{discoverError}</p>
        </div>
      )}

      {vers.length === 0 ? (
        <div className="mt-4 rounded-xl border border-border bg-bg-elevated px-6 py-8 text-center">
          <p className="text-sm text-text-secondary">版本发现未返回任何结果。</p>
          <p className="mt-2 text-sm text-text-muted">
            可能是该语言没有可访问的官网下载页，或网络原因导致抓取失败。可尝试重新添加。
          </p>
        </div>
      ) : (
        <div className="mt-4 divide-y divide-border rounded-xl border border-border">
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
                  <KnowledgeStatusBadge kbStatus={v.kb_status} />
                  {v.image && (
                    <span className="font-mono text-xs text-text-muted">{v.image}</span>
                  )}
                </div>
                <VersionActions
                  version={v}
                  initializingId={initializingId}
                  buildingId={buildingId}
                  initError={initErrors[v.id]}
                  languageId={lang.id}
                  languageActive={lang.status === 'active'}
                  onToggleStatus={() => handleToggleStatus(v.id, v.status)}
                  onInitialize={() => handleInitialize(v.id)}
                  onCancelInit={handleCancelInit}
                  onBuildKnowledge={() => handleBuildKnowledge(v.id)}
                  onStopPolling={() => handleStopPolling(v.id)}
                  onToggleKB={() => handleToggleKB(v.id)}
                  onOpenPlayground={() => setActiveVersionId(v.id)}
                  expandedKB={expandedKB === v.id}
                  loadingKB={loadingKB === v.id}
                />
              </div>

              {/* Inline playground */}
              {activeVersionId === v.id && (
                <PlaygroundPanel
                  versionId={v.id}
                  languageName={lang.name}
                  versionLabel={v.version}
                  languageActive={lang.status === 'active'}
                  onClose={() => setActiveVersionId('')}
                />
              )}

              {/* Knowledge base */}
              {expandedKB === v.id && kbEntries[v.id] && (
                <KnowledgeBasePanel entries={kbEntries[v.id]} />
              )}

              {/* Loading KB */}
              {loadingKB === v.id && (
                <div className="border-t border-border bg-bg-base/50 px-6 py-4">
                  <p className="text-sm text-text-muted">加载知识库…</p>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
