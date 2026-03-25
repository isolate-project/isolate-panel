import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/preact'
import { Dashboard } from '../../pages/Dashboard'
import { statsApi } from '../../api/endpoints'

vi.mock('../../api/endpoints', () => ({
  statsApi: {
    getDashboardStats: vi.fn(),
  },
}))

describe('Dashboard Page', () => {
  const mockStats = {
    data: {
      success: true,
      stats: {
        total_users: 150,
        active_users: 120,
        online_users: 45,
        total_inbounds: 25,
        total_traffic_used_bytes: 1099511627776,
        total_traffic_limit_bytes: 10995116277760,
        cores_running: 2,
        cores_total: 3,
      },
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(statsApi.getDashboardStats).mockResolvedValue(mockStats)
  })

  it('renders dashboard page', async () => {
    render(<Dashboard />)
    expect(screen.getByText(/dashboard.title/i)).toBeInTheDocument()
  })

  it('displays total users stat', async () => {
    render(<Dashboard />)
    await waitFor(() => {
      expect(screen.getByText(/150/i)).toBeInTheDocument()
    })
  })

  it('displays active users stat', async () => {
    render(<Dashboard />)
    await waitFor(() => {
      expect(screen.getByText(/120/i)).toBeInTheDocument()
    })
  })

  it('displays online users stat', async () => {
    render(<Dashboard />)
    await waitFor(() => {
      expect(screen.getByText(/45/i)).toBeInTheDocument()
    })
  })

  it('displays traffic usage', async () => {
    render(<Dashboard />)
    await waitFor(() => {
      expect(screen.getByText(/1.0 TB/i)).toBeInTheDocument()
    })
  })

  it('displays cores status', async () => {
    render(<Dashboard />)
    await waitFor(() => {
      expect(screen.getByText(/2\/3/i)).toBeInTheDocument()
    })
  })
})
