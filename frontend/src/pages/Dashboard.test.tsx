import { render, screen } from '@testing-library/preact'
import { describe, it, expect, vi } from 'vitest'
import { Dashboard } from './Dashboard'

// Mock the hooks used in Dashboard
vi.mock('../hooks/useUsers', () => ({
  useUsers: () => ({ data: { users: [] }, isLoading: false })
}))
vi.mock('../hooks/useCores', () => ({
  useCores: () => ({ data: [], isLoading: false })
}))
vi.mock('../hooks/useSystem', () => ({
  useSystemResources: () => ({ data: { cpu: { percent: 10 }, ram: { percent: 20, used: 100, total: 500 } } })
}))
vi.mock('../hooks/useConnections', () => ({
  useConnections: () => ({ count: 0, isLoading: false })
}))
vi.mock('preact-router', () => ({
  route: vi.fn(),
}))

describe('Dashboard Component', () => {
  it('renders dashboard UI correctly with mock data', () => {
    render(<Dashboard />)
    
    expect(screen.getByText('dashboard.title')).toBeInTheDocument()
    expect(screen.getByText('dashboard.totalUsers')).toBeInTheDocument()
    expect(screen.getByText('dashboard.systemResources')).toBeInTheDocument()
  })
})
