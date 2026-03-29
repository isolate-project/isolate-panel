import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook } from '@testing-library/preact'
import { useSessionExpired } from './useSessionExpired'
import { useToastStore } from '../stores/toastStore'

// Mock toast store
vi.mock('../stores/toastStore', () => ({
  useToastStore: () => ({
    addToast: vi.fn(),
  }),
}))

describe('useSessionExpired Hook', () => {
  const mockAddToast = useToastStore().addToast

  beforeEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('should trigger toast when accessToken is removed from storage', () => {
    renderHook(() => useSessionExpired())

    const storageEvent = new StorageEvent('storage', {
      key: 'accessToken',
      oldValue: 'some-token',
      newValue: null
    })

    window.dispatchEvent(storageEvent)

    expect(mockAddToast).toHaveBeenCalledWith({
      type: 'warning',
      message: expect.any(String),
      duration: 5000
    })
  })

  it('should not trigger toast multiple times in short interval', () => {
    renderHook(() => useSessionExpired())

    const storageEvent = new StorageEvent('storage', {
      key: 'accessToken',
      oldValue: 'some-token',
      newValue: null
    })

    window.dispatchEvent(storageEvent)
    window.dispatchEvent(storageEvent)

    expect(mockAddToast).toHaveBeenCalledTimes(1)
  })

  it('should reset debounce flag after timeout', () => {
    vi.useFakeTimers()
    renderHook(() => useSessionExpired())

    const storageEvent = new StorageEvent('storage', {
      key: 'accessToken',
      oldValue: 'some-token',
      newValue: null
    })

    window.dispatchEvent(storageEvent)
    expect(mockAddToast).toHaveBeenCalledTimes(1)

    // Advance time beyond 6000ms debounce
    vi.advanceTimersByTime(6001)

    window.dispatchEvent(storageEvent)
    expect(mockAddToast).toHaveBeenCalledTimes(2)

    vi.useRealTimers()
  })

  it('should cleanup timeout on unmount', () => {
    vi.useFakeTimers()
    const clearSpy = vi.spyOn(window, 'clearTimeout')
    const { unmount } = renderHook(() => useSessionExpired())

    // Trigger one
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'accessToken',
      oldValue: 't1',
      newValue: null
    }))

    unmount()
    expect(clearSpy).toHaveBeenCalled()
    
    vi.useRealTimers()
  })
})
