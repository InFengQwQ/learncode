import Badge from '../ui/Badge'

export default function KnowledgeStatusBadge({ kbStatus }: { kbStatus: string }) {
  switch (kbStatus) {
    case 'complete':
      return <Badge variant="success">知识库就绪</Badge>
    case 'building':
      return <Badge variant="info">构建中</Badge>
    case 'failed':
      return <Badge variant="danger">构建失败</Badge>
    default:
      return <Badge variant="warning">待构建</Badge>
  }
}
