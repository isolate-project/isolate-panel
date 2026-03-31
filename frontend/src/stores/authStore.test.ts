import { describe, it, expect, beforeEach, vi } from 'vitest'
vi.unmock('./authStore')
import { useAuthStore } from './authStore'

describe('authStore', () => {
  beforeEach(() => {
    localStorage.clear()
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isLoading: false
    })
  })

  it('should initialize with default state', () => {
    const state = useAuthStore.getState()
    expect(state.accessToken).toBeNull()
    expect(state.isAuthenticated).toBe(false)
  })

  it('should set tokens and update localStorage', () => {
    useAuthStore.getState().setTokens('access123', 'refresh456')
    const state = useAuthStore.getState()
    
    expect(state.accessToken).toBe('access123')
    expect(state.refreshToken).toBe('refresh456')
    expect(state.isAuthenticated).toBe(true)
    
    expect(localStorage.getItem('accessToken')).toBe('access123')
    expect(localStorage.getItem('refreshToken')).toBe('refresh456')
  })

  it('should set user', () => {
    const user = { id: 1, username: 'admin', is_super_admin: true }
    useAuthStore.getState().setUser(user)
    expect(useAuthStore.getState().user).toEqual(user)
  })

  it('should logout and clear localStorage', () => {
    useAuthStore.getState().setTokens('access123', 'refresh456')
    useAuthStore.getState().logout()
    
    const state = useAuthStore.getState()
    expect(state.accessToken).toBeNull()
    expect(state.isAuthenticated).toBe(false)
    
    expect(localStorage.getItem('accessToken')).toBeNull()
    expect(localStorage.getItem('refreshToken')).toBeNull()
  })
})
