import type { LanguageVersion } from '../../api/client'
import Button from '../ui/Button'
import BuildKnowledgeButton from './BuildKnowledgeButton'

interface VersionActionsProps {
  version: LanguageVersion
  initializingId: string
  buildingId: string
  initError?: string
  languageId: string
  languageActive: boolean
  onToggleStatus: () => void
  onInitialize: () => void
  onCancelInit: () => void
  onBuildKnowledge: () => void
  onStopPolling: () => void
  onToggleKB: () => void
  onOpenPlayground: () => void
  expandedKB: boolean
  loadingKB: boolean
}

export default function VersionActions({
  version: v,
  initializingId,
  buildingId,
  initError,
  languageActive,
  onToggleStatus,
  onInitialize,
  onCancelInit,
  onBuildKnowledge,
  onStopPolling,
  onToggleKB,
  onOpenPlayground,
  expandedKB,
  loadingKB,
}: VersionActionsProps) {
  return (
    <div className="flex items-center gap-2 shrink-0">
      {/* Archive / Activate toggle */}
      <Button variant="ghost" size="sm" onClick={onToggleStatus}>
        {v.status === 'active' ? '归档' : '激活'}
      </Button>

      {/* Initialize button */}
      {!v.initialized &&
        (initializingId === v.id ? (
          <Button variant="ghost" size="sm" onClick={onCancelInit}>
            取消初始化
          </Button>
        ) : (
          <Button variant="ghost" size="sm" onClick={onInitialize}>
            初始化环境
          </Button>
        ))}

      {/* Build knowledge button */}
      {v.initialized && (
        <BuildKnowledgeButton
          kbStatus={v.kb_status}
          buildingId={buildingId}
          versionId={v.id}
          onBuild={onBuildKnowledge}
          onStopPolling={onStopPolling}
        />
      )}

      {/* Init error */}
      {initError && (
        <span className="text-xs text-danger">{initError}</span>
      )}

      {/* View knowledge base */}
      {v.initialized && v.kb_status === 'complete' && (
        <Button variant="ghost" size="sm" onClick={onToggleKB}>
          {expandedKB
            ? '收起知识库'
            : loadingKB
              ? '加载中…'
              : '查看知识库'}
        </Button>
      )}

      {/* Run button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={onOpenPlayground}
        disabled={!v.initialized || !languageActive}
      >
        {!languageActive ? '未激活' : '运行'}
      </Button>
    </div>
  )
}
