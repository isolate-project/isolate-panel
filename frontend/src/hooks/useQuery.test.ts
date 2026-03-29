import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/preact'
import { useQuery } from './useQuery'
import { cache } from '../utils/cache'

describe('useQuery Hook', () => {
  const mockData = { id: 1, name: 'Test' }
  const mockFetcher = vi.fn().mockResolvedValue(mockData)

  beforeEach(() => {
    cache.clear()
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('should fetch data on mount', async () => {
    const { result } = renderHook(() => useQuery('test-key', mockFetcher))

    expect(result.current.isLoading).toBe(true)
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
      expect(result.current.data).toEqual(mockData)
    })

    expect(mockFetcher).toHaveBeenCalled()
    expect(cache.get('test-key')).toEqual(mockData)
  })

  it('should return cached data immediately if available', async () => {
    cache.set('cached-key', { id: 2, name: 'Cached' })
    
    const { result } = renderHook(() => useQuery('cached-key', mockFetcher))

    expect(result.current.isLoading).toBe(false)
    expect(result.current.data).toEqual({ id: 2, name: 'Cached' })
    
    // It shouldn't fetch if cached
    expect(mockFetcher).not.toHaveBeenCalled()
  })

  it('should handle errors', async () => {
    const errorFetcher = vi.fn().mockRejectedValue(new Error('Fetch failed'))
    const { result } = renderHook(() => useQuery('error-key', errorFetcher))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
      expect(result.current.error?.message).toBe('Fetch failed')
    })
  })

  it('should respect the enabled option', () => {
    renderHook(() => useQuery('disabled-key', mockFetcher, { enabled: false }))
    expect(mockFetcher).not.toHaveBeenCalled()
  })

  it('should support polling with refetchInterval', async () => {
    vi.useFakeTimers()
    const pollFetcher = vi.fn().mockResolvedValue({ status: 'ok' })
    
    renderHook(() => useQuery('poll-key', pollFetcher, { refetchInterval: 1000 }))

    await waitFor(() => {
      expect(pollFetcher).toHaveBeenCalledTimes(1)
    })

    vi.advanceTimersByTime(1001)
    expect(pollFetcher).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(1001)
    expect(pollFetcher).toHaveBeenCalledTimes(3)
  })

  it('should abort previous requests on refetch', async () => {
    const abortSpy = vi.spyOn(AbortController.prototype, 'abort')
    const { result } = renderHook(() => useQuery('abort-key', mockFetcher))

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    
    await result.current.refetch()
    expect(abortSpy).toHaveBeenCalled()
  })

  it('should update state only if mounted', async () => {
    let resolvePromise: (value: any) => void
    const longFetcher = vi.fn().mockImplementation(() => new Promise((resolve) => {
      resolvePromise = resolve
    }))

    const { unmount } = renderHook(() => useQuery('mount-test', longFetcher))
    
    unmount()
    resolvePromise!({ data: 'late' })
    
    // Should not throw or cause state updates on unmounted component
    // (verified by not seeing "warning: update on unmounted component")
  })
})
