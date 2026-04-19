import { useMutation } from './useMutation'
import { inboundApi } from '../api/endpoints'

export interface CheckPortParams {
  port: number
  listen?: string
  protocol: string
  transport?: string
  core_type: string
}

export interface PortConflict {
  inbound_id: number
  inbound_name: string
  protocol: string
  transport?: string
  core_type: string
  port: number
  haproxy_compatible: boolean
  can_share: boolean
  sharing_mechanism?: string
  requires_confirm: boolean
}

export interface PortValidationResult {
  port: number
  listen_address: string
  protocol: string
  transport?: string
  core_type: string
  is_available: boolean
  haproxy_compatible: boolean
  can_share_port: boolean
  sharing_mechanism?: string
  severity: 'info' | 'warning' | 'error'
  action: 'allow' | 'confirm' | 'block'
  message: string
  conflicts?: PortConflict[]
}

export function useCheckPort() {
  return useMutation<PortValidationResult, CheckPortParams>(
    async (params) => {
      const response = await inboundApi.checkPortAvailability({
        port: params.port,
        listen: params.listen || '0.0.0.0',
        protocol: params.protocol,
        transport: params.transport || '',
        core_type: params.core_type,
      })
      return response.data as PortValidationResult
    }
  )
}
