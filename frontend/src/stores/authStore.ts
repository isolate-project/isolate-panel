import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface User {
  id: number
  username: string
  is_super_admin: boolean
  must_change_password: boolean
}

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean

  setTokens: (accessToken: string, refreshToken: string) => void
  setUser: (user: User | null) => void
  logout: () => void
  setLoading: (loading: boolean) => void
  clearMustChangePassword: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isLoading: false,

      setTokens: (accessToken, refreshToken) => {
        // Single source of truth: Zustand persist writes to localStorage under 'auth-storage'.
        // We also write dedicated keys so the API client interceptor can read them
        // without importing the store (avoids circular dependency).
        localStorage.setItem('accessToken', accessToken)
        localStorage.setItem('refreshToken', refreshToken)
        set({ accessToken, refreshToken, isAuthenticated: true })
      },

      setUser: (user) => set({ user }),

      logout: () => {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        set({
          accessToken: null,
          refreshToken: null,
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
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
      // On rehydration, sync dedicated localStorage keys
      onRehydrateStorage: () => (state) => {
        if (state?.accessToken) {
          localStorage.setItem('accessToken', state.accessToken)
        }
        if (state?.refreshToken) {
          localStorage.setItem('refreshToken', state.refreshToken)
        }
      },
    }
  )
)
