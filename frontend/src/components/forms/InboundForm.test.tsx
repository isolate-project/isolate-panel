import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/preact'
import { InboundForm } from './InboundForm'
import type { Inbound, Core, ProtocolSummary } from '../../types'

vi.mock('../../hooks/useCores')
vi.mock('../../hooks/useProtocols')
vi.mock('../../hooks/useInbounds')
vi.mock('../../hooks/useQuery')
vi.mock('../../hooks/useCheckPort')
vi.mock('../inbound/PortValidationField', () => ({
  PortValidationField: ({ value, onChange, disabled }: { value: number; onChange: (port: number) => void; disabled?: boolean }) => (
    <input
      type="number"
      value={value || ''}
      onChange={(e: Event) => {
        const target = e.target as HTMLInputElement
        const newPort = parseInt(target.value, 10)
        onChange(isNaN(newPort) ? 0 : newPort)
      }}
      disabled={disabled}
      data-testid="port-input"
    />
  ),
}))

import { useCores } from '../../hooks/useCores'
import { useProtocols } from '../../hooks/useProtocols'
import { useCreateInbound, useUpdateInbound } from '../../hooks/useInbounds'
import { useQuery } from '../../hooks/useQuery'
import { useCheckPort } from '../../hooks/useCheckPort'

const mockCores: Core[] = [
  {
    id: 1,
    name: 'xray-core',
    type: 'xray',
    version: '1.8.0',
    is_enabled: true,
    is_running: true,
    pid: 12345,
    uptime_seconds: 3600,
    restart_count: 0,
    last_error: '',
  },
  {
    id: 2,
    name: 'singbox-core',
    type: 'sing-box',
    version: '1.9.0',
    is_enabled: true,
    is_running: true,
    pid: 12346,
    uptime_seconds: 3600,
    restart_count: 0,
    last_error: '',
  },
]

const mockProtocols: ProtocolSummary[] = [
  {
    protocol: 'vless',
    label: 'VLESS',
    description: 'VLESS protocol',
    core: ['xray', 'sing-box'],
    direction: 'both',
    requires_tls: false,
    category: 'standard',
  },
  {
    protocol: 'vmess',
    label: 'VMess',
    description: 'VMess protocol',
    core: ['xray'],
    direction: 'both',
    requires_tls: false,
    category: 'standard',
    deprecated: true,
    deprecation_notice: 'VMess is deprecated due to security concerns. Please migrate to VLESS.',
  },
  {
    protocol: 'trojan',
    label: 'Trojan',
    description: 'Trojan protocol',
    core: ['xray', 'sing-box'],
    direction: 'both',
    requires_tls: true,
    category: 'standard',
  },
  {
    protocol: 'hysteria',
    label: 'Hysteria',
    description: 'Hysteria protocol',
    core: ['sing-box'],
    direction: 'inbound',
    requires_tls: false,
    category: 'standard',
    deprecated: true,
    deprecation_notice: 'Hysteria is deprecated. Please use Hysteria2 instead.',
  },
  {
    protocol: 'shadowsocks',
    label: 'Shadowsocks',
    description: 'Shadowsocks protocol',
    core: ['sing-box'],
    direction: 'both',
    requires_tls: false,
    category: 'standard',
  },
]

const mockExistingInbound: Inbound = {
  id: 1,
  name: 'Test Inbound',
  protocol: 'vless',
  core_id: 1,
  listen_address: '0.0.0.0',
  port: 443,
  config_json: '{}',
  tls_enabled: true,
  tls_cert_id: null,
  reality_enabled: false,
  reality_config_json: '{}',
  is_enabled: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  core: {
    id: 1,
    name: 'xray-core',
    type: 'xray',
  },
}

describe('InboundForm', () => {
  const mockOnSuccess = vi.fn()
  const mockOnCancel = vi.fn()
  const mockCreateInbound = vi.fn()
  const mockUpdateInbound = vi.fn()
  const mockCheckPort = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(useCores).mockReturnValue({
      data: mockCores,
      error: null,
      isLoading: false,
      isRefetching: false,
      refetch: vi.fn(),
    })

    vi.mocked(useProtocols).mockReturnValue({
      data: { protocols: mockProtocols, total: mockProtocols.length },
      error: null,
      isLoading: false,
      isRefetching: false,
      refetch: vi.fn(),
    })

    vi.mocked(useCreateInbound).mockReturnValue({
      mutate: mockCreateInbound,
      isLoading: false,
      error: null,
      data: null,
      reset: vi.fn(),
    })

    vi.mocked(useUpdateInbound).mockReturnValue({
      mutate: mockUpdateInbound,
      isLoading: false,
      error: null,
      data: null,
      reset: vi.fn(),
    })

    vi.mocked(useQuery).mockReturnValue({
      data: { options: [] },
      error: null,
      isLoading: false,
      isRefetching: false,
      refetch: vi.fn(),
    })

    vi.mocked(useCheckPort).mockReturnValue({
      mutate: mockCheckPort,
      isLoading: false,
      error: null,
      data: null,
      reset: vi.fn(),
    })
  })

  describe('renders form fields for new inbound', () => {
    it('renders all required form fields', () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      expect(screen.getByLabelText(/inbounds.name/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/inbounds.protocol/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/inbounds.core/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/inbounds.listenAddress/i)).toBeInTheDocument()
      expect(screen.getByTestId('port-input')).toBeInTheDocument()
      expect(screen.getByText(/Enable TLS Encryption/i)).toBeInTheDocument()
      expect(screen.getByText(/Enable Inbound Connection/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Create Inbound/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /common.cancel/i })).toBeInTheDocument()
    })
  })

  describe('filters protocols by selected core type', () => {
    it('shows only xray protocols when xray core is selected', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      const protocolOptions = protocolSelect.querySelectorAll('option')
      const protocolValues = Array.from(protocolOptions).map(opt => opt.value)

      expect(protocolValues).toContain('vless')
      expect(protocolValues).toContain('vmess')
      expect(protocolValues).toContain('trojan')
      expect(protocolValues).not.toContain('hysteria')
      expect(protocolValues).not.toContain('shadowsocks')
    })

    it('shows only sing-box protocols when sing-box core is selected', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const coreSelect = screen.getByLabelText(/inbounds.core/i)
      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      fireEvent.change(coreSelect, { target: { value: '2' } })

      await waitFor(() => {
        const protocolOptions = protocolSelect.querySelectorAll('option')
        const protocolValues = Array.from(protocolOptions).map(opt => opt.value)

        expect(protocolValues).toContain('vless')
        expect(protocolValues).toContain('trojan')
        expect(protocolValues).toContain('hysteria')
        expect(protocolValues).toContain('shadowsocks')
        expect(protocolValues).not.toContain('vmess')
      })
    })
  })

  describe('shows deprecated warning when deprecated protocol selected', () => {
    it('displays warning for deprecated hysteria protocol', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const coreSelect = screen.getByLabelText(/inbounds.core/i)
      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      fireEvent.change(coreSelect, { target: { value: '2' } })

      await waitFor(() => {
        fireEvent.change(protocolSelect, { target: { value: 'hysteria' } })
      })

      await waitFor(() => {
        const warning = screen.getByText(/⚠ Deprecated:/i)
        expect(warning).toBeInTheDocument()
        expect(warning.parentElement).toHaveTextContent(/Hysteria is deprecated. Please use Hysteria2 instead./i)
      })
    })

    it('displays warning for deprecated vmess protocol', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      fireEvent.change(protocolSelect, { target: { value: 'vmess' } })

      await waitFor(() => {
        const warning = screen.getByText(/⚠ Deprecated:/i)
        expect(warning).toBeInTheDocument()
        expect(warning.parentElement).toHaveTextContent(/VMess is deprecated due to security concerns. Please migrate to VLESS./i)
      })
    })
  })

  describe('hides deprecated warning for non-deprecated protocol', () => {
    it('does not show warning for vless protocol', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      fireEvent.change(protocolSelect, { target: { value: 'vless' } })

      await waitFor(() => {
        const warning = screen.queryByText(/⚠ Deprecated:/i)
        expect(warning).not.toBeInTheDocument()
      })
    })

    it('does not show warning for trojan protocol', async () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const protocolSelect = screen.getByLabelText(/inbounds.protocol/i)

      fireEvent.change(protocolSelect, { target: { value: 'trojan' } })

      await waitFor(() => {
        const warning = screen.queryByText(/⚠ Deprecated:/i)
        expect(warning).not.toBeInTheDocument()
      })
    })
  })

  describe('calls createInbound on submit for new inbound', () => {
    it('submits form and calls createInbound mutation', async () => {
      mockCreateInbound.mockResolvedValueOnce(undefined)

      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const nameInput = screen.getByLabelText(/inbounds.name/i)
      fireEvent.change(nameInput, { target: { value: 'Test Inbound' } })

      const portInput = screen.getByTestId('port-input')
      fireEvent.change(portInput, { target: { value: '8443' } })

      const submitButton = screen.getByRole('button', { name: /Create Inbound/i })
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(mockCreateInbound).toHaveBeenCalledTimes(1)
        expect(mockCreateInbound).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'Test Inbound',
            port: 8443,
          })
        )
      })

      await waitFor(() => {
        expect(mockOnSuccess).toHaveBeenCalledTimes(1)
      })
    })
  })

  describe('calls updateInbound on submit for existing inbound', () => {
    it('submits form and calls updateInbound mutation', async () => {
      mockUpdateInbound.mockResolvedValueOnce(undefined)

      render(
        <InboundForm
          inbound={mockExistingInbound}
          onSuccess={mockOnSuccess}
          onCancel={mockOnCancel}
        />
      )

      const nameInput = screen.getByLabelText(/inbounds.name/i)
      fireEvent.change(nameInput, { target: { value: 'Updated Inbound' } })

      const submitButton = screen.getByRole('button', { name: /Save Changes/i })
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(mockUpdateInbound).toHaveBeenCalledTimes(1)
        expect(mockUpdateInbound).toHaveBeenCalledWith({
          id: mockExistingInbound.id,
          data: expect.objectContaining({
            name: 'Updated Inbound',
          }),
        })
      })

      await waitFor(() => {
        expect(mockOnSuccess).toHaveBeenCalledTimes(1)
      })
    })

    it('shows Save Changes button for existing inbound', () => {
      render(
        <InboundForm
          inbound={mockExistingInbound}
          onSuccess={mockOnSuccess}
          onCancel={mockOnCancel}
        />
      )

      expect(screen.getByRole('button', { name: /Save Changes/i })).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Create Inbound/i })).not.toBeInTheDocument()
    })
  })

  describe('calls onCancel when cancel button clicked', () => {
    it('triggers onCancel callback when cancel button is clicked', () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const cancelButton = screen.getByRole('button', { name: /common.cancel/i })
      fireEvent.click(cancelButton)

      expect(mockOnCancel).toHaveBeenCalledTimes(1)
    })

    it('does not call createInbound when cancel is clicked', () => {
      render(<InboundForm onSuccess={mockOnSuccess} onCancel={mockOnCancel} />)

      const cancelButton = screen.getByRole('button', { name: /common.cancel/i })
      fireEvent.click(cancelButton)

      expect(mockCreateInbound).not.toHaveBeenCalled()
    })
  })
})