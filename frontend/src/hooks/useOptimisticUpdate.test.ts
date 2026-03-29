import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/preact'
import { useOptimisticUpdate } from './useOptimisticUpdate'
import { cache } from '../utils/cache'

describe('useOptimisticUpdate Hook', () => {
  const queryKey = 'test-data'
  const originalData = { id: 1, name: 'Original' }
  const mutationFn = vi.fn().mockImplementation((newData) => Promise.resolve(newData))

  beforeEach(() => {
    cache.clear()
    cache.set(queryKey, originalData)
    vi.clearAllMocks()
  })

  it('should optimistically update data in cache', async () => {
    const { result } = renderHook(() => useOptimisticUpdate(queryKey, mutationFn))
    
    // Mutation is not yet sent
    expect(cache.get(queryKey)).toEqual(originalData)

    act(() => {
        result.current.mutate({ id: 1, name: 'Optimistic' })
    })

    // Should immediately update cache
    expect(cache.get(queryKey)).toEqual({ id: 1, name: 'Optimistic' })

    await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
    })
  })

  it('should rollback data on mutation error', async () => {
    const errorMutation = vi.fn().mockRejectedValue(new Error('Update failed'))
    const { result } = renderHook(() => useOptimisticUpdate(queryKey, errorMutation))
    
    act(() => {
        result.current.mutate({ id: 1, name: 'Fail' })
    })

    // Optimistic update
    expect(cache.get(queryKey)).toEqual({ id: 1, name: 'Fail' })

    await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
    })

    // Rollback
    expect(cache.get(queryKey)).toEqual(originalData)
    expect(result.current.error?.message).toBe('Update failed')
  })
})
