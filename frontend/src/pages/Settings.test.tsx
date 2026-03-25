import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/preact'
import { Settings } from '../../pages/Settings'
import { systemApi } from '../../api/endpoints'

// Mock systemApi
vi.mock('../../api/endpoints', () => ({
  systemApi: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    getMonitoring: vi.fn(),
    updateMonitoring: vi.fn(),
  },
}))

describe('Settings Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    
    // Mock successful API responses
    vi.mocked(systemApi.getSettings).mockResolvedValue({
      data: {
        panel_name: 'Isolate Panel',
        jwt_access_token_ttl: 900,
        jwt_refresh_token_ttl: 604800,
        max_login_attempts: 5,
        log_level: 'info',
      },
    })
    
    vi.mocked(systemApi.getMonitoring).mockResolvedValue({
      data: {
        success: true,
        mode: 'lite',
        interval: 60,
      },
    })
  })

  it('renders settings page', async () => {
    render(<Settings />)
    
    // Check for page title
    expect(screen.getByText(/settings.title/i)).toBeInTheDocument()
    
    // Wait for loading to complete
    await waitFor(() => {
      expect(screen.getByText(/settings.general/i)).toBeInTheDocument()
    })
  })

  it('renders appearance section', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      expect(screen.getByText(/settings.appearance/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.theme/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.language/i)).toBeInTheDocument()
    })
  })

  it('renders general settings section', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      expect(screen.getByText(/settings.general/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.panelName/i)).toBeInTheDocument()
    })
  })

  it('renders security settings section', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      expect(screen.getByText(/settings.security/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.jwtTokenTTL/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.maxLoginAttempts/i)).toBeInTheDocument()
    })
  })

  it('renders monitoring mode section', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      expect(screen.getByText(/settings.monitoringMode/i)).toBeInTheDocument()
      expect(screen.getByText(/settings.monitoringModeLabel/i)).toBeInTheDocument()
    })
  })

  it('loads settings from API', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      expect(systemApi.getSettings).toHaveBeenCalledTimes(1)
      expect(systemApi.getMonitoring).toHaveBeenCalledTimes(1)
    })
  })

  it('displays panel name from API', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      const panelNameInput = screen.getByDisplayValue('Isolate Panel')
      expect(panelNameInput).toBeInTheDocument()
    })
  })

  it('displays monitoring mode from API', async () => {
    render(<Settings />)
    
    await waitFor(() => {
      const monitoringSelect = screen.getByRole('combobox', { name: /settings.monitoringModeLabel/i })
      expect(monitoringSelect).toHaveValue('lite')
    })
  })

  it('handles monitoring mode change', async () => {
    vi.mocked(systemApi.updateMonitoring).mockResolvedValue({ data: { success: true } })
    
    render(<Settings />)
    
    await waitFor(() => {
      const monitoringSelect = screen.getByRole('combobox', { name: /settings.monitoringModeLabel/i })
      fireEvent.change(monitoringSelect, { target: { value: 'full' } })
    })
    
    await waitFor(() => {
      expect(systemApi.updateMonitoring).toHaveBeenCalledWith({ mode: 'full' })
    })
  })

  it('saves settings on button click', async () => {
    vi.mocked(systemApi.updateSettings).mockResolvedValue({ data: { success: true } })
    
    render(<Settings />)
    
    await waitFor(() => {
      const saveButton = screen.getByText(/common.save/i)
      fireEvent.click(saveButton)
    })
  })
})
