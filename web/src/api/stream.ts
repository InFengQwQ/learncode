export async function* readNDJSONStream(
  response: Response,
  signal?: AbortSignal,
): AsyncGenerator<Record<string, unknown>> {
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: `HTTP ${response.status}` }))
    yield { step: 'fatal', status: 'error', message: (err as Record<string, string>).error ?? `HTTP ${response.status}` }
    return
  }

  const reader = response.body!.getReader()
  if (signal) {
    signal.addEventListener('abort', () => reader.cancel(), { once: true })
  }

  const decoder = new TextDecoder()
  let buf = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split('\n')
      buf = lines.pop() ?? ''
      for (const line of lines) {
        if (!line.trim()) continue
        try {
          yield JSON.parse(line)
        } catch {
          // skip malformed lines
        }
      }
    }
  } catch (e) {
    if (signal?.aborted) return
    throw e
  }

  if (buf.trim()) {
    try {
      yield JSON.parse(buf)
    } catch {
      // skip
    }
  }
}
