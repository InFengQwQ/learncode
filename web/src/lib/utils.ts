export function externalURL(raw: string): string {
  if (!raw) return ''
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw
  if (raw.startsWith('//')) return 'https:' + raw
  return 'https://' + raw
}

export function friendlyError(err: string): string {
  if (err.includes('language not found')) return '未找到此名称的编程语言'
  if (err.includes('not a programming language')) return '这不是一门编程语言'
  if (err.includes('invalid compatibility_model')) return '兼容性模型无效，请联系管理员'
  if (err.includes('invalid slug')) return '标识符格式无效，请重新查询'
  if (err.includes('invalid docs_url')) return '文档链接无效，请重新查询'
  if (err.includes('invalid runtime_url')) return '运行时链接无效，请重新查询'
  return err
}

// isIconURL returns true when the icon value is an HTTP(S) URL (from Wikipedia
// page image) rather than an emoji string.
export function isIconURL(icon: string): boolean {
  return icon.startsWith('http://') || icon.startsWith('https://')
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
