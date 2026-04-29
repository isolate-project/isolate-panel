import { create } from 'zustand'

interface User {
  id: number
  username: string
  is_super_admin: boolean
  must_change_password: boolean
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean

  setUser: (user: User | null) => void
  setAuthenticated: (authenticated: boolean) => void
  logout: () => void
  setLoading: (loading: boolean) => void
  clearMustChangePassword: () => void
}

export const useAuthStore = create<AuthState>()(
  (set) => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,

    setUser: (user) => set({ user }),

    setAuthenticated: (authenticated) => set({ isAuthenticated: authenticated }),

    logout: () => {
      set({
        user: null,
        isAuthenticated: false,
      })
    },

    setLoading: (loading) => set({ isLoading: loading }),

    clearMustChangePassword: () => {
      set((state) => ({
        user: state.user ? { ...state.user, must_change_password: false } : null,
      }))
    },
  })
)
