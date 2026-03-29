import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  message: string
  duration?: number
}

interface ToastState {
  toasts: Toast[]
  timeouts: Map<string, number>
  addToast: (toast: Omit<Toast, 'id'>) => void
  removeToast: (id: string) => void
  clearAll: () => void
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  timeouts: new Map(),
  
  addToast: (toast) => {
    const id = Math.random().toString(36).substring(2, 9)
    const newToast: Toast = {
      ...toast,
      id,
      duration: toast.duration || 5000,
    }

    set((state) => ({
      toasts: [...state.toasts, newToast],
    }))

    // Auto-remove after duration
    if (newToast.duration && newToast.duration > 0) {
      const timeoutId = window.setTimeout(() => {
        get().removeToast(id)
      }, newToast.duration)
      get().timeouts.set(id, timeoutId)
    }
  },

  removeToast: (id) => {
    const timeoutId = get().timeouts.get(id)
    if (timeoutId) {
      window.clearTimeout(timeoutId)
      get().timeouts.delete(id)
    }
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }))
  },

  clearAll: () => {
    get().timeouts.forEach((id) => window.clearTimeout(id))
    get().timeouts.clear()
    set({ toasts: [] })
  },
}))
