import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/preact'
import { afterEach } from 'vitest'

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
  useAuthStore: () => ({
    token: 'mock-token',
    setToken: () => {},
    logout: () => {},
  }),
}))
