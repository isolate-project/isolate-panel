import { describe, it, expect, beforeEach, vi } from 'vitest'
vi.unmock('./themeStore')
import { useThemeStore } from './themeStore'

describe('themeStore', () => {
  beforeEach(() => {
    useThemeStore.setState({ theme: 'light' })
    document.documentElement.className = ''
    document.documentElement.removeAttribute('data-theme')
  })

  it('should default to light theme', () => {
    expect(useThemeStore.getState().theme).toBe('light')
  })

  it('should set theme to dark and modify DOM', () => {
    useThemeStore.getState().setTheme('dark')
    expect(useThemeStore.getState().theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('should toggle theme', () => {
    useThemeStore.getState().toggleTheme()
    expect(useThemeStore.getState().theme).toBe('dark')
    
    useThemeStore.getState().toggleTheme()
    expect(useThemeStore.getState().theme).toBe('light')
  })
})
