import { useState, useEffect, useRef, useCallback } from 'react'

export interface UseApiState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

export interface UseApiOptions {
  fetchOnMount?: boolean
}

export function useApi<T>(
  fetcher: (signal: AbortSignal) => Promise<{ ok: boolean; data?: T; error?: string }>,
  options: UseApiOptions = {},
) {
  const { fetchOnMount = true } = options
  const [state, setState] = useState<UseApiState<T>>({
    data: null,
    loading: false,
    error: null,
  })
  const abortRef = useRef<AbortController | null>(null)
  const mountedRef = useRef(true)
  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  const execute = useCallback(async () => {
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    setState((prev) => ({ ...prev, loading: true, error: null }))

    try {
      const res = await fetcherRef.current(controller.signal)
      if (!mountedRef.current || controller.signal.aborted) return
      if (res.ok) {
        setState({ data: (res.data ?? null) as T | null, loading: false, error: null })
      } else {
        setState({ data: null, loading: false, error: res.error ?? '未知错误' })
      }
    } catch (e) {
      if (!mountedRef.current || controller.signal.aborted) return
      setState({ data: null, loading: false, error: e instanceof Error ? e.message : '未知错误' })
    }
  }, [])

  const refetch = useCallback(() => {
    return execute()
  }, [execute])

  useEffect(() => {
    mountedRef.current = true
    if (fetchOnMount) {
      execute()
    }
    return () => {
      mountedRef.current = false
      abortRef.current?.abort()
    }
  }, [fetchOnMount]) // eslint-disable-line react-hooks/exhaustive-deps

  return { ...state, refetch }
}
