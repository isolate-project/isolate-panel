import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/preact'
import { Backups } from '../../pages/Backups'
import { backupApi } from '../../api/endpoints'

vi.mock('../../api/endpoints', () => ({
  backupApi: {
    listBackups: vi.fn(),
    createBackup: vi.fn(),
    deleteBackup: vi.fn(),
  },
}))

describe('Backups Page', () => {
  const mockBackups = {
    data: {
      success: true,
      backups: [
        {
          id: 1,
          name: 'backup-20260325.db',
          created_at: '2026-03-25T02:00:00Z',
          size_bytes: 1048576,
        },
      ],
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(backupApi.listBackups).mockResolvedValue(mockBackups)
  })

  it('renders backups page', async () => {
    render(<Backups />)
    expect(screen.getByText(/backups.title/i)).toBeInTheDocument()
  })

  it('displays create backup button', async () => {
    render(<Backups />)
    await waitFor(() => {
      expect(screen.getByText(/backups.createBackup/i)).toBeInTheDocument()
    })
  })

  it('loads backups from API', async () => {
    render(<Backups />)
    await waitFor(() => {
      expect(backupApi.listBackups).toHaveBeenCalledTimes(1)
    })
  })

  it('displays backup list', async () => {
    render(<Backups />)
    await waitFor(() => {
      expect(screen.getByText(/backup-20260325.db/i)).toBeInTheDocument()
    })
  })

  it('displays backup size', async () => {
    render(<Backups />)
    await waitFor(() => {
      expect(screen.getByText(/1.0 MB/i)).toBeInTheDocument()
    })
  })

  it('displays backup date', async () => {
    render(<Backups />)
    await waitFor(() => {
      expect(screen.getByText(/2026-03-25/i)).toBeInTheDocument()
    })
  })
})
