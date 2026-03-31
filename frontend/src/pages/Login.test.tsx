import { render, screen } from '@testing-library/preact'
import { describe, it, expect, vi } from 'vitest'
import { Login } from './Login'

// Mock preact-router
vi.mock('preact-router', () => ({
  route: vi.fn(),
}))

describe('Login Component', () => {
  it('renders login form elements properly', () => {
    render(<Login />)
    
    // The translated strings are checked via the mock ('auth.loginButton', etc.)
    expect(screen.getByText('common.appName')).toBeInTheDocument()
    expect(screen.getByText('auth.loginButton')).toBeInTheDocument()
  })
})
