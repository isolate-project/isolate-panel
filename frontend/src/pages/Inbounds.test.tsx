import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/preact'
import { Inbounds } from '../../pages/Inbounds'
import { inboundsApi } from '../../api/endpoints'

vi.mock('../../api/endpoints', () => ({
  inboundsApi: {
    listInbounds: vi.fn(),
    deleteInbound: vi.fn(),
  },
}))

describe('Inbounds Page', () => {
  const mockInbounds = {
    data: {
      success: true,
      inbounds: [
        {
          id: 1,
          name: 'VMess-443',
          protocol: 'vmess',
          port: 443,
          is_enabled: true,
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 2,
          name: 'VLESS-8443',
          protocol: 'vless',
          port: 8443,
          is_enabled: false,
          created_at: '2026-02-01T00:00:00Z',
        },
      ],
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(inboundsApi.listInbounds).mockResolvedValue(mockInbounds)
  })

  it('renders inbounds page', async () => {
    render(<Inbounds />)
    expect(screen.getByText(/inbounds.title/i)).toBeInTheDocument()
  })

  it('displays add inbound button', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText(/inbounds.addInbound/i)).toBeInTheDocument()
    })
  })

  it('loads inbounds from API', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(inboundsApi.listInbounds).toHaveBeenCalledTimes(1)
    })
  })

  it('displays inbounds list', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText('VMess-443')).toBeInTheDocument()
      expect(screen.getByText('VLESS-8443')).toBeInTheDocument()
    })
  })

  it('displays protocol information', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText(/vmess/i)).toBeInTheDocument()
      expect(screen.getByText(/vless/i)).toBeInTheDocument()
    })
  })

  it('displays port information', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText('443')).toBeInTheDocument()
      expect(screen.getByText('8443')).toBeInTheDocument()
    })
  })

  it('displays enabled status', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText(/enabled/i)).toBeInTheDocument()
      expect(screen.getByText(/disabled/i)).toBeInTheDocument()
    })
  })

  it('displays search input', async () => {
    render(<Inbounds />)
    await waitFor(() => {
      const searchInput = screen.getByPlaceholderText(/inbounds.searchPlaceholder/i)
      expect(searchInput).toBeInTheDocument()
    })
  })

  it('handles inbound deletion', async () => {
    vi.mocked(inboundsApi.deleteInbound).mockResolvedValue({ data: { success: true } })
    render(<Inbounds />)
    await waitFor(() => {
      const deleteButtons = screen.getAllByRole('button', { name: /delete/i })
      if (deleteButtons.length > 0) {
        fireEvent.click(deleteButtons[0])
      }
    })
  })

  it('shows loading state initially', () => {
    render(<Inbounds />)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('handles empty inbounds list', async () => {
    vi.mocked(inboundsApi.listInbounds).mockResolvedValue({
      data: { success: true, inbounds: [] },
    })
    render(<Inbounds />)
    await waitFor(() => {
      expect(screen.getByText(/inbounds.noInbounds/i)).toBeInTheDocument()
    })
  })
})
