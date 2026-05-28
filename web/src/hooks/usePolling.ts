import { useRef, useCallback, useState, useEffect } from 'react'

export interface UsePollingOptions<T> {
  interval: number
  maxAttempts?: number
  enabled?: boolean
  onComplete?: (data: T) => void
  onError?: (error: string) => void
}

export function usePolling<T>(
  fetcher: () => Promise<{ done: boolean; data?: T; error?: string }>,
  options: UsePollingOptions<T>,
) {
  const { interval, maxAttempts = 30, enabled = false, onComplete, onError } = options
  const [isPolling, setIsPolling] = useState(false)
  const [attempts, setAttempts] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const mountedRef = useRef(true)

  const stop = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
    setIsPolling(false)
  }, [])

  const start = useCallback(() => {
    stop()
    setAttempts(0)
    setIsPolling(true)
  }, [stop])

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      stop()
    }
  }, [stop])

  useEffect(() => {
    if (!enabled && !isPolling) return

    let cancelled = false

    const poll = async () => {
      if (cancelled) return

      try {
        const result = await fetcher()
        if (!mountedRef.current || cancelled) return

        setAttempts((prev) => prev + 1)

        if (result.done) {
          stop()
          if (result.data) onComplete?.(result.data)
        } else if (attempts >= maxAttempts - 1) {
          stop()
          onError?.('轮询超时')
        }
      } catch {
        if (!mountedRef.current || cancelled) return
        stop()
        onError?.('轮询失败')
      }
    }

    if (isPolling || enabled) {
      poll()
      intervalRef.current = setInterval(poll, interval)
    }

    return () => {
      cancelled = true
    }
  }, [enabled, isPolling, interval, maxAttempts, fetcher, stop, onComplete, onError, attempts])

  return { start, stop, isPolling, attempts }
}
