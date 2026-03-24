import { useState, useEffect, useCallback } from 'preact/hooks'
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
  const { enabled = true, refetchInterval, cacheTime = 300000, onSuccess, onError } = options

  const [data, setData] = useState<T | null>(() => cache.get(key))
  const [error, setError] = useState<Error | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isRefetching, setIsRefetching] = useState(false)

  const fetchData = useCallback(
    async (isRefetch = false) => {
      if (!enabled) return

      // Check cache first (only on initial load, not refetch)
      if (!isRefetch) {
        const cachedData = cache.get<T>(key)
        if (cachedData) {
          setData(cachedData)
          setIsLoading(false)
          return
        }
      }

      try {
        if (isRefetch) {
          setIsRefetching(true)
        } else {
          setIsLoading(true)
        }

        const result = await fetcher()
        cache.set(key, result, cacheTime)
        setData(result)
        setError(null)
        onSuccess?.(result)
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Unknown error')
        setError(error)
        onError?.(error)
      } finally {
        setIsLoading(false)
        setIsRefetching(false)
      }
    },
    [enabled, fetcher, key, cacheTime, onSuccess, onError]
  )

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Polling
  useEffect(() => {
    if (!refetchInterval || !enabled) return

    const interval = setInterval(() => {
      fetchData(true)
    }, refetchInterval)

    return () => clearInterval(interval)
  }, [refetchInterval, enabled, fetchData])

  const refetch = useCallback(() => fetchData(true), [fetchData])

  return { data, error, isLoading, isRefetching, refetch }
}
