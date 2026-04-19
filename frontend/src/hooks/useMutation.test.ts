import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/preact'
import { useMutation } from './useMutation'

describe('useMutation Hook', () => {
  const mockMutationFn = vi.fn().mockImplementation((data) => Promise.resolve({ success: true, ...data }))

  beforeEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('should call mutation function and update state on success', async () => {
    const { result } = renderHook(() => useMutation(mockMutationFn))

    expect(result.current.isLoading).toBe(false)
    
    act(() => {
      result.current.mutate({ test: 'data' })
    })

    expect(result.current.isLoading).toBe(true)
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
      expect(result.current.data).toEqual({ success: true, test: 'data' })
    })

    expect(mockMutationFn).toHaveBeenCalledWith({ test: 'data' }, expect.any(AbortSignal))
  })

  it('should update state on error', async () => {
    const errorMutationFn = vi.fn().mockRejectedValue(new Error('Mutation failed'))
    const { result } = renderHook(() => useMutation(errorMutationFn))

    act(() => {
      result.current.mutate({ some: 'data' }).catch(() => {})
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
      expect(result.current.error?.message).toBe('Mutation failed')
    })
  })

  it('should reset mutation state', async () => {
    const { result } = renderHook(() => useMutation(mockMutationFn))

    act(() => {
      result.current.mutate({ data: 'to reset' })
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.data).toBeDefined()

    act(() => {
      result.current.reset()
    })

    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('should call callbacks on success and error', async () => {
    const onSuccess = vi.fn()
    const onError = vi.fn()
    const { result } = renderHook(() => useMutation(mockMutationFn, { onSuccess, onError }))

    act(() => {
      result.current.mutate({ key: 'val' })
    })

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith({ success: true, key: 'val' }, { key: 'val' })
    })
    
    // Test error callback
    const errorFn = vi.fn().mockRejectedValue(new Error('Fail'))
    const { result: errResult } = renderHook(() => useMutation(errorFn, { onSuccess, onError }))

    act(() => {
      errResult.current.mutate({ key2: 'val2' }).catch(() => {})
    })

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(expect.any(Error), { key2: 'val2' })
    })
  })

  it('should not update state if unmounted', async () => {
    let resolvePromise: (value: unknown) => void
    const longMutation = vi.fn().mockImplementation(() => new Promise((resolve) => {
      resolvePromise = resolve
    }))

    const { result, unmount } = renderHook(() => useMutation(longMutation))
    
    act(() => {
      result.current.mutate({ data: 'late' })
    })
    
    unmount()
    resolvePromise!({ data: 'late' })
    
    // Should not trigger errors for updating unmounted component
  })
})
