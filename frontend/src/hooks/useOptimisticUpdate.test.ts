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
    const { result } = renderHook(() => useOptimisticUpdate<typeof originalData>(queryKey))
    
    // Mutation is not yet sent
    expect(cache.get(queryKey)).toEqual(originalData)

    act(() => {
        result.current.optimisticUpdate(
            () => ({ id: 1, name: 'Optimistic' }),
            () => mutationFn({ id: 1, name: 'Optimistic' })
        ).catch(() => {})
    })

    // Should immediately update cache
    expect(cache.get(queryKey)).toEqual({ id: 1, name: 'Optimistic' })
  })

  it('should rollback data on mutation error', async () => {
    const errorMutation = vi.fn().mockRejectedValue(new Error('Update failed'))
    const { result } = renderHook(() => useOptimisticUpdate<typeof originalData>(queryKey))
    
    act(() => {
        result.current.optimisticUpdate(
            () => ({ id: 1, name: 'Fail' }),
            () => errorMutation()
        ).catch(() => {})
    })

    // Optimistic update
    expect(cache.get(queryKey)).toEqual({ id: 1, name: 'Fail' })

    // Rollback hasn't happened yet unless we wait for the unhandled promise rejection to process in microtasks
    await waitFor(() => {
        expect(cache.get(queryKey)).toEqual(originalData)
    })
  })
})
