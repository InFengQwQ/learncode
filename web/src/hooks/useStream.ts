import { useRef, useCallback, useState } from 'react'

export interface UseStreamOptions {
  onLine?: (line: Record<string, unknown>) => void
  onDone?: () => void
  onError?: (error: string) => void
}

export function useStream(options: UseStreamOptions = {}) {
  const [status, setStatus] = useState<'idle' | 'streaming' | 'done' | 'error'>('idle')
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const start = useCallback(
    async (
      streamFactory: (signal: AbortSignal) => AsyncGenerator<Record<string, unknown>>,
    ) => {
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller

      setStatus('streaming')
      setError(null)

      let hadProgress = false

      try {
        for await (const line of streamFactory(controller.signal)) {
          if (controller.signal.aborted) return 'cancelled' as const

          hadProgress = true

          if ('ok' in line && line.ok) {
            options.onLine?.(line)
            setStatus('done')
            options.onDone?.()
            return 'done' as const
          }

          if (line.step === 'fatal' && line.status === 'error') {
            const msg = (line.message as string) ?? '未知错误'
            setError(msg)
            setStatus('error')
            options.onError?.(msg)
            return 'error' as const
          }

          options.onLine?.(line)
        }

        if (!hadProgress) {
          setStatus('done')
          return 'done' as const
        }

        setStatus('done')
        options.onDone?.()
        return 'done' as const
      } catch (e) {
        if (controller.signal.aborted) {
          setStatus('idle')
          return 'cancelled' as const
        }
        const msg = e instanceof Error ? e.message : '流式传输失败'
        setError(msg)
        setStatus('error')
        options.onError?.(msg)
        return 'error' as const
      }
    },
    [options],
  )

  const cancel = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setStatus('idle')
  }, [])

  return { status, error, start, cancel }
}
