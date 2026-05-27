export function externalURL(raw: string): string {
  if (!raw) return ''
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw
  if (raw.startsWith('//')) return 'https:' + raw
  return 'https://' + raw
}

export function friendlyError(err: string): string {
  // Chinese step-prefixed errors from backend — return as-is (already localized).
  if (err.startsWith('Wikipedia') || err.startsWith('LLM') || err.startsWith('未找到'))
    return err

  // Legacy English errors.
  if (err.includes('language not found')) return '未找到此名称的编程语言'
  if (err.includes('not a programming language')) return '这不是一门编程语言'
  if (err.includes('no Wikipedia article')) return '在 Wikipedia 中未找到此名称'
  if (err.includes('strict language') && err.includes('no versions')) return '未能发现任何版本，无法创建严格模式语言'
  if (err.includes('invalid compatibility_model')) return '兼容性模型无效'
  if (err.includes('invalid slug')) return '标识符格式无效'
  if (err.includes('network')) return '网络连接失败，请检查网络后重试'
  if (err.includes('timeout')) return '查询超时，请稍后重试'
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
