import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/preact'
import { afterEach, vi } from 'vitest'

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: {
      language: 'en',
      changeLanguage: () => Promise.resolve(),
    },
  }),
  initReactI18next: {
    type: '3rdParty',
    init: () => {},
  },
}))

// Mock zustand stores
vi.mock('../stores/themeStore', () => ({
  useThemeStore: () => ({
    theme: 'light',
    setTheme: () => {},
  }),
}))

vi.mock('../stores/toastStore', () => ({
  useToastStore: () => ({
    addToast: () => {},
  }),
}))

vi.mock('../stores/authStore', () => ({
  useAuthStore: Object.assign(
    () => ({
      accessToken: 'mock-token',
      refreshToken: 'mock-refresh-token',
      user: { id: 1, username: 'admin', is_super_admin: true },
      isAuthenticated: true,
      isLoading: false,
      setTokens: () => {},
      setUser: () => {},
      logout: () => {},
      setLoading: () => {},
    }),
    {
      getState: () => ({
        accessToken: 'mock-token',
        refreshToken: 'mock-refresh-token',
        user: { id: 1, username: 'admin', is_super_admin: true },
        isAuthenticated: true,
        isLoading: false,
        setTokens: () => {},
        setUser: () => {},
        logout: () => {},
        setLoading: () => {},
      }),
    }
  ),
}))

// Mock lucide-preact - automatic mock for all icons
vi.mock('lucide-preact', async () => {
  const actual = await vi.importActual('lucide-preact')
  const mockComponent = () => null
  
  const mocks: Record<string, unknown> = {}
  Object.keys(actual).forEach(key => {
    mocks[key] = mockComponent
  })
  
  return mocks
})
