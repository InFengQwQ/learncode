import Button from '../ui/Button'

interface BuildKnowledgeButtonProps {
  kbStatus: string
  buildingId: string
  versionId: string
  onBuild: () => void
  onStopPolling: () => void
}

export default function BuildKnowledgeButton({
  kbStatus,
  buildingId,
  versionId,
  onBuild,
  onStopPolling,
}: BuildKnowledgeButtonProps) {
  const isBuilding = kbStatus === 'building'
  const needsBuild = !kbStatus || kbStatus === 'pending' || kbStatus === 'failed'

  if (isBuilding) {
    return (
      <Button variant="ghost" size="sm" onClick={onStopPolling}>
        停止监控
      </Button>
    )
  }

  if (!needsBuild) return null

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={onBuild}
      disabled={buildingId === versionId}
    >
      {buildingId === versionId
        ? '启动中…'
        : kbStatus === 'failed'
          ? '重试构建'
          : '构建知识库'}
    </Button>
  )
}
