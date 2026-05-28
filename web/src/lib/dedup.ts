const inflight = new Map<string, Promise<unknown>>()

function dedupKey(path: string, options?: RequestInit): string {
  return `${options?.method ?? 'GET'}:${path}:${options?.body ?? ''}`
}

export async function requestDeduped<T>(
  fetcher: () => Promise<T>,
  path: string,
  options?: RequestInit,
): Promise<T> {
  const key = dedupKey(path, options)
  const existing = inflight.get(key)
  if (existing) return existing as Promise<T>

  const promise = fetcher()
  inflight.set(key, promise)

  try {
    return await promise
  } finally {
    inflight.delete(key)
  }
}
