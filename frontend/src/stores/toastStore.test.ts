import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useToastStore } from './toastStore'

describe('Toast Store', () => {
  beforeEach(() => {
    useToastStore.getState().clearAll()
  })

  it('should add a toast with a unique ID', () => {
    const store = useToastStore.getState()
    store.addToast({ type: 'success', message: 'Hello' })
    
    const state = useToastStore.getState()
    expect(state.toasts.length).toBe(1)
    expect(state.toasts[0].message).toBe('Hello')
    expect(state.toasts[0].id).toBeDefined()
  })

  it('should remove a specific toast', () => {
    const store = useToastStore.getState()
    store.addToast({ type: 'success', message: 'Toast 1' })
    const toastId = useToastStore.getState().toasts[0].id
    
    useToastStore.getState().removeToast(toastId)
    expect(useToastStore.getState().toasts.length).toBe(0)
  })

  it('should track and clear timeouts when removing a toast', () => {
    vi.useFakeTimers()
    const spy = vi.spyOn(window, 'clearTimeout')
    
    const store = useToastStore.getState()
    store.addToast({ type: 'success', message: 'T1', duration: 1000 })
    const toastId = useToastStore.getState().toasts[0].id
    
    useToastStore.getState().removeToast(toastId)
    
    expect(spy).toHaveBeenCalled()
    expect(useToastStore.getState().timeouts.has(toastId)).toBe(false)
    
    vi.useRealTimers()
  })

  it('should auto-remove toast after duration', () => {
    vi.useFakeTimers()
    
    const store = useToastStore.getState()
    store.addToast({ type: 'success', message: 'Auto-remove', duration: 1000 })
    
    expect(useToastStore.getState().toasts.length).toBe(1)
    
    vi.advanceTimersByTime(1001)
    
    expect(useToastStore.getState().toasts.length).toBe(0)
    
    vi.useRealTimers()
  })

  it('should clear all toasts and timeouts', () => {
    vi.useFakeTimers()
    const spy = vi.spyOn(window, 'clearTimeout')
    
    const store = useToastStore.getState()
    store.addToast({ type: 'success', message: 'T1', duration: 1000 })
    store.addToast({ type: 'info', message: 'T2', duration: 1000 })
    
    useToastStore.getState().clearAll()
    
    expect(useToastStore.getState().toasts.length).toBe(0)
    expect(useToastStore.getState().timeouts.size).toBe(0)
    expect(spy).toHaveBeenCalledTimes(2)
    
    vi.useRealTimers()
  })
})
