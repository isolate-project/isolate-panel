import { useState, useEffect, useCallback, useRef } from 'preact/hooks'
import { cache } from '../utils/cache'

interface UseQueryOptions<T> {
  enabled?: boolean
  refetchInterval?: number
  cacheTime?: number
  onSuccess?: (data: T) => void
  onError?: (error: Error) => void
}

interface UseQueryResult<T> {
  data: T | null
  error: Error | null
  isLoading: boolean
  isRefetching: boolean
  refetch: () => Promise<void>
}

export function useQuery<T>(
  key: string,
  fetcher: () => Promise<T>,
  options: UseQueryOptions<T> = {}
): UseQueryResult<T> {
  const { enabled = true, refetchInterval, cacheTime = 300000 } = options
  
  const hasLoadedRef = useRef(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const intervalIdRef = useRef<number | null>(null)
  const mountedRef = useRef(true)

  // Store fetcher and callbacks in refs to avoid dependency instability
  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher
  const onSuccessRef = useRef(options.onSuccess)
  onSuccessRef.current = options.onSuccess
  const onErrorRef = useRef(options.onError)
  onErrorRef.current = options.onError

  const [data, setData] = useState<T | null>(() => cache.get(key))
  const [error, setError] = useState<Error | null>(null)
  const [isLoading, setIsLoading] = useState(!cache.get(key))
  const [isRefetching, setIsRefetching] = useState(false)

  // Track mounted state
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      // Abort any pending request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      // Clear any interval
      if (intervalIdRef.current !== null) {
        clearInterval(intervalIdRef.current)
      }
    }
  }, [])

  const fetchData = useCallback(async (isRefetch = false) => {
    if (!enabled) return

    // Abort previous request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    // Create new abort controller
    abortControllerRef.current = new AbortController()
    const signal = abortControllerRef.current.signal

    // Check cache first (only on initial load)
    if (!isRefetch && !hasLoadedRef.current) {
      const cachedData = cache.get<T>(key)
      if (cachedData) {
        setData(cachedData)
        setIsLoading(false)
        hasLoadedRef.current = true
        onSuccessRef.current?.(cachedData)
        return
      }
    }

    try {
      if (isRefetch) {
        setIsRefetching(true)
      } else {
        setIsLoading(true)
      }

      const result = await fetcherRef.current()
      
      // Don't update state if aborted or unmounted
      if (signal.aborted || !mountedRef.current) {
        return
      }
      
      cache.set(key, result, cacheTime)
      setData(result)
      setError(null)
      hasLoadedRef.current = true
      onSuccessRef.current?.(result)
    } catch (err) {
      // Ignore abort errors
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }
      // Don't update state if unmounted
      if (!mountedRef.current) {
        return
      }
      
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      onErrorRef.current?.(error)
    } finally {
      if (!signal.aborted && mountedRef.current) {
        setIsLoading(false)
        setIsRefetching(false)
      }
    }
  }, [enabled, key, cacheTime])

  const prevKeyRef = useRef<string>(key)

  // Initial load or key change
  useEffect(() => {
    if (enabled && (!hasLoadedRef.current || prevKeyRef.current !== key)) {
      prevKeyRef.current = key
      fetchData()
    }
  }, [enabled, key, fetchData])

  // Polling - only if refetchInterval is set
  useEffect(() => {
    if (!refetchInterval || !enabled) {
      // Clear existing interval if disabled
      if (intervalIdRef.current !== null) {
        clearInterval(intervalIdRef.current)
        intervalIdRef.current = null
      }
      return
    }

    // Clear any existing interval
    if (intervalIdRef.current !== null) {
      clearInterval(intervalIdRef.current)
    }

    // Set up new interval
    intervalIdRef.current = window.setInterval(() => {
      fetchData(true)
    }, refetchInterval)

    // Cleanup
    return () => {
      if (intervalIdRef.current !== null) {
        clearInterval(intervalIdRef.current)
        intervalIdRef.current = null
      }
    }
  }, [refetchInterval, enabled, fetchData])

  const refetch = useCallback(() => fetchData(true), [fetchData])

  return { data, error, isLoading, isRefetching, refetch }
}
