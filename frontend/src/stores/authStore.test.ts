import { describe, it, expect, beforeEach, vi } from 'vitest'
vi.unmock('./authStore')
import { useAuthStore } from './authStore'

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: false
    })
  })

  it('should initialize with default state', () => {
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.user).toBeNull()
  })

  it('should set authenticated', () => {
    useAuthStore.getState().setAuthenticated(true)
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(true)
  })

  it('should set user', () => {
    const user = { id: 1, username: 'admin', is_super_admin: true, must_change_password: false }
    useAuthStore.getState().setUser(user)
    expect(useAuthStore.getState().user).toEqual(user)
  })

  it('should logout', () => {
    useAuthStore.getState().setAuthenticated(true)
    useAuthStore.getState().logout()
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.user).toBeNull()
  })
})
