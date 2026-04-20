import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/preact'
import { SubscriptionLinks } from './SubscriptionLinks'
import type { User } from '../../types'
import { useSubscriptionStats, useRegenerateToken } from '../../hooks/useSubscriptionStats'

vi.mock('../../hooks/useSubscriptionStats', () => ({
  useSubscriptionStats: vi.fn(),
  useRegenerateToken: vi.fn(),
}))

describe('SubscriptionLinks', () => {
  const mockUser: User = {
    id: 1,
    username: 'testuser',
    email: 'test@example.com',
    uuid: '123e4567-e89b-12d3-a456-426614174000',
    subscription_token: 'test-token-abc123',
    traffic_limit_bytes: 10737418240,
    traffic_used_bytes: 536870912,
    expiry_date: '2025-12-31T23:59:59Z',
    is_active: true,
    is_online: false,
    created_at: '2025-01-01T00:00:00Z',
    last_connected_at: null,
  }

  const mockUserNoToken: User = {
    ...mockUser,
    subscription_token: '',
  }

  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useSubscriptionStats).mockReturnValue({
      data: null,
      error: null,
      isLoading: false,
      isRefetching: false,
      refetch: vi.fn(),
    })
    vi.mocked(useRegenerateToken).mockReturnValue({
      mutate: vi.fn(),
      isLoading: false,
      error: null,
      data: null,
      reset: vi.fn(),
    })
  })

  it('renders all 4 subscription links', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    expect(screen.getByText('subscriptions.v2rayLink')).toBeInTheDocument()
    expect(screen.getByText('subscriptions.clashLink')).toBeInTheDocument()
    expect(screen.getByText('subscriptions.singboxLink')).toBeInTheDocument()
    expect(screen.getByText('subscriptions.isolateLink')).toBeInTheDocument()
  })

  it('renders correct URLs for each format', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    const baseUrl = window.location.origin
    const token = mockUser.subscription_token

    expect(screen.getByText(`${baseUrl}/sub/${token}`)).toBeInTheDocument()
    expect(screen.getByText(`${baseUrl}/sub/${token}/clash`)).toBeInTheDocument()
    expect(screen.getByText(`${baseUrl}/sub/${token}/singbox`)).toBeInTheDocument()
    expect(screen.getByText(`${baseUrl}/sub/${token}/isolate`)).toBeInTheDocument()
  })

  it('renders isolate link with correct URL', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    const baseUrl = window.location.origin
    const token = mockUser.subscription_token
    const expectedIsolateUrl = `${baseUrl}/sub/${token}/isolate`

    const isolateUrl = screen.getByText(expectedIsolateUrl)
    expect(isolateUrl).toBeInTheDocument()

    expect(screen.getByText('subscriptions.isolateLink')).toBeInTheDocument()
  })

  it('shows no token message when user has no subscription token', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUserNoToken} />)

    expect(screen.getByText('subscriptions.noToken')).toBeInTheDocument()

    expect(screen.queryByText('subscriptions.v2rayLink')).not.toBeInTheDocument()
    expect(screen.queryByText('subscriptions.clashLink')).not.toBeInTheDocument()
    expect(screen.queryByText('subscriptions.singboxLink')).not.toBeInTheDocument()
    expect(screen.queryByText('subscriptions.isolateLink')).not.toBeInTheDocument()
  })

  it('each link has copy and external link buttons', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    const baseUrl = window.location.origin
    const token = mockUser.subscription_token

    const allButtons = screen.getAllByRole('button')
    const copyButtons = allButtons.filter(button => {
      const hasGhostVariant = button.classList.contains('bg-transparent') || button.classList.contains('hover:bg-hover')
      return hasGhostVariant
    })
    expect(copyButtons.length).toBeGreaterThanOrEqual(4)

    const externalLinks = screen.getAllByRole('link').filter(link => link.hasAttribute('target'))
    expect(externalLinks.length).toBeGreaterThanOrEqual(4)

    const expectedUrls = [
      `${baseUrl}/sub/${token}`,
      `${baseUrl}/sub/${token}/clash`,
      `${baseUrl}/sub/${token}/singbox`,
      `${baseUrl}/sub/${token}/isolate`,
    ]

    const actualHrefs = externalLinks.map(link => link.getAttribute('href'))
    expectedUrls.forEach(url => {
      expect(actualHrefs).toContain(url)
    })
  })

  it('renders action buttons when user has token', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    expect(screen.getByText('subscriptions.showQR')).toBeInTheDocument()
    expect(screen.getByText('subscriptions.viewStats')).toBeInTheDocument()
    expect(screen.getByText('subscriptions.regenerate')).toBeInTheDocument()
  })

  it('does not render action buttons when user has no token', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUserNoToken} />)

    expect(screen.queryByText('subscriptions.showQR')).not.toBeInTheDocument()
    expect(screen.queryByText('subscriptions.viewStats')).not.toBeInTheDocument()
    expect(screen.queryByText('subscriptions.regenerate')).not.toBeInTheDocument()
  })

  it('renders modal title', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    expect(screen.getByText('subscriptions.title')).toBeInTheDocument()
  })

  it('renders modal description', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    expect(screen.getByText('subscriptions.description')).toBeInTheDocument()
  })

  it('renders close button', () => {
    render(<SubscriptionLinks isOpen={true} onClose={mockOnClose} user={mockUser} />)

    const closeButton = screen.getByRole('button', { name: /common.close/i })
    expect(closeButton).toBeInTheDocument()

    fireEvent.click(closeButton)
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })
})