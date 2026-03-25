import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/preact'
import { Notifications } from '../../pages/Notifications'
import { notificationApi } from '../../api/endpoints'

vi.mock('../../api/endpoints', () => ({
  notificationApi: {
    listNotifications: vi.fn(),
    markAsRead: vi.fn(),
  },
}))

describe('Notifications Page', () => {
  const mockNotifications = {
    data: {
      success: true,
      notifications: [
        {
          id: 1,
          type: 'user_created',
          message: 'User testuser created',
          is_read: false,
          created_at: '2026-03-25T10:00:00Z',
        },
      ],
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(notificationApi.listNotifications).mockResolvedValue(mockNotifications)
  })

  it('renders notifications page', async () => {
    render(<Notifications />)
    expect(screen.getByText(/notifications.title/i)).toBeInTheDocument()
  })

  it('loads notifications from API', async () => {
    render(<Notifications />)
    await waitFor(() => {
      expect(notificationApi.listNotifications).toHaveBeenCalledTimes(1)
    })
  })

  it('displays notification list', async () => {
    render(<Notifications />)
    await waitFor(() => {
      expect(screen.getByText(/User testuser created/i)).toBeInTheDocument()
    })
  })

  it('displays unread indicator', async () => {
    render(<Notifications />)
    await waitFor(() => {
      expect(screen.getByText(/notifications.unread/i)).toBeInTheDocument()
    })
  })

  it('marks notification as read', async () => {
    vi.mocked(notificationApi.markAsRead).mockResolvedValue({ data: { success: true } })
    render(<Notifications />)
    await waitFor(() => {
      const markReadBtn = screen.getByRole('button', { name: /mark as read/i })
      if (markReadBtn) {
        fireEvent.click(markReadBtn)
      }
    })
  })
})
