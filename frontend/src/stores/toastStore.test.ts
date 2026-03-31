import { describe, it, expect, beforeEach, vi } from 'vitest'
import { toastStore } from './toastStore'

vi.unmock('./toastStore')
vi.unmock('../stores/toastStore')

describe('Toast Store', () => {
  beforeEach(() => {
    toastStore.getState().clearAll()
  })

  it('should add a toast with a unique ID', () => {
    toastStore.getState().addToast({ type: 'success', message: 'Hello' })
    
    const state = toastStore.getState()
    expect(state.toasts.length).toBe(1)
    expect(state.toasts[0].message).toBe('Hello')
    expect(state.toasts[0].id).toBeDefined()
  })

  it('should remove a specific toast', () => {
    toastStore.getState().addToast({ type: 'success', message: 'Toast 1' })
    const toastId = toastStore.getState().toasts[0].id
    
    toastStore.getState().removeToast(toastId)
    expect(toastStore.getState().toasts.length).toBe(0)
  })

  it('should track and clear timeouts when removing a toast', () => {
    vi.useFakeTimers()
    const spy = vi.spyOn(window, 'clearTimeout')
    
    toastStore.getState().addToast({ type: 'success', message: 'T1', duration: 1000 })
    const toastId = toastStore.getState().toasts[0].id
    
    toastStore.getState().removeToast(toastId)
    
    expect(spy).toHaveBeenCalled()
    expect(toastStore.getState().timeouts.has(toastId)).toBe(false)
    
    vi.useRealTimers()
  })

  it('should auto-remove toast after duration', () => {
    vi.useFakeTimers()
    
    toastStore.getState().addToast({ type: 'success', message: 'Auto-remove', duration: 1000 })
    
    expect(toastStore.getState().toasts.length).toBe(1)
    
    vi.advanceTimersByTime(1001)
    
    expect(toastStore.getState().toasts.length).toBe(0)
    
    vi.useRealTimers()
  })

  it('should clear all toasts and timeouts', () => {
    vi.useFakeTimers()
    const spy = vi.spyOn(window, 'clearTimeout')
    
    toastStore.getState().addToast({ type: 'success', message: 'T1', duration: 1000 })
    toastStore.getState().addToast({ type: 'info', message: 'T2', duration: 1000 })
    
    toastStore.getState().clearAll()
    
    expect(toastStore.getState().toasts.length).toBe(0)
    expect(toastStore.getState().timeouts.size).toBe(0)
    expect(spy).toHaveBeenCalledTimes(2)
    
    vi.useRealTimers()
  })
})
