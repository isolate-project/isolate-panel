import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/preact'
import { Users } from '../../pages/Users'
import { usersApi } from '../../api/endpoints'

// Mock usersApi
vi.mock('../../api/endpoints', () => ({
  usersApi: {
    listUsers: vi.fn(),
    deleteUser: vi.fn(),
  },
}))

describe('Users Page', () => {
  const mockUsers = {
    data: {
      success: true,
      users: [
        {
          id: 1,
          username: 'user1',
          email: 'user1@example.com',
          uuid: 'test-uuid-1',
          is_active: true,
          traffic_limit_bytes: 107374182400,
          traffic_used_bytes: 10737418240,
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 2,
          username: 'user2',
          email: 'user2@example.com',
          uuid: 'test-uuid-2',
          is_active: false,
          traffic_limit_bytes: 53687091200,
          traffic_used_bytes: 0,
          created_at: '2026-02-01T00:00:00Z',
        },
      ],
      total: 2,
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(usersApi.listUsers).mockResolvedValue(mockUsers)
  })

  it('renders users page', async () => {
    render(<Users />)
    
    // Check for page title
    expect(screen.getByText(/users.title/i)).toBeInTheDocument()
  })

  it('renders add user button', async () => {
    render(<Users />)
    
    await waitFor(() => {
      expect(screen.getByText(/users.addUser/i)).toBeInTheDocument()
    })
  })

  it('loads users from API', async () => {
    render(<Users />)
    
    await waitFor(() => {
      expect(usersApi.listUsers).toHaveBeenCalledTimes(1)
    })
  })

  it('displays users list', async () => {
    render(<Users />)
    
    await waitFor(() => {
      expect(screen.getByText('user1')).toBeInTheDocument()
      expect(screen.getByText('user2')).toBeInTheDocument()
    })
  })

  it('displays user emails', async () => {
    render(<Users />)
    
    await waitFor(() => {
      expect(screen.getByText('user1@example.com')).toBeInTheDocument()
      expect(screen.getByText('user2@example.com')).toBeInTheDocument()
    })
  })

  it('displays user status', async () => {
    render(<Users />)
    
    await waitFor(() => {
      // Active user
      expect(screen.getByText(/active/i)).toBeInTheDocument()
      // Inactive user
      expect(screen.getByText(/inactive/i)).toBeInTheDocument()
    })
  })

  it('displays traffic usage', async () => {
    render(<Users />)
    
    await waitFor(() => {
      // Should show traffic information
      expect(screen.getByText(/10.0 GB/i)).toBeInTheDocument()
    })
  })

  it('displays search input', async () => {
    render(<Users />)
    
    await waitFor(() => {
      const searchInput = screen.getByPlaceholderText(/users.searchPlaceholder/i)
      expect(searchInput).toBeInTheDocument()
    })
  })

  it('handles user deletion', async () => {
    vi.mocked(usersApi.deleteUser).mockResolvedValue({ data: { success: true } })
    
    render(<Users />)
    
    await waitFor(() => {
      // Find and click delete button for first user
      const deleteButtons = screen.getAllByRole('button', { name: /delete/i })
      if (deleteButtons.length > 0) {
        fireEvent.click(deleteButtons[0])
      }
    })
  })

  it('shows loading state initially', () => {
    render(<Users />)
    
    // Should show spinner or loading indicator
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('handles empty users list', async () => {
    vi.mocked(usersApi.listUsers).mockResolvedValue({
      data: { success: true, users: [], total: 0 },
    })
    
    render(<Users />)
    
    await waitFor(() => {
      expect(screen.getByText(/users.noUsers/i)).toBeInTheDocument()
    })
  })
})
