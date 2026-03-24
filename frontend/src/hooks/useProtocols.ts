import { useQuery } from './useQuery'
import { protocolApi } from '../api/endpoints'
import type { ProtocolSummary, ProtocolSchema, ProtocolDefaults } from '../types'

// List all protocols with optional filters
export function useProtocols(params?: { core?: string; direction?: string }) {
  return useQuery<{ protocols: ProtocolSummary[]; total: number }>(
    `protocols-${params?.core || 'all'}-${params?.direction || 'all'}`,
    () => protocolApi.list(params).then((res) => res.data),
    { cacheTime: 600000 } // 10 min cache — protocols rarely change
  )
}

// Get full protocol schema
export function useProtocolSchema(name: string) {
  return useQuery<ProtocolSchema>(
    `protocol-schema-${name}`,
    () => protocolApi.get(name).then((res) => res.data),
    { enabled: !!name, cacheTime: 600000 }
  )
}

// Get protocol defaults (auto-generated values)
export function useProtocolDefaults(name: string) {
  return useQuery<ProtocolDefaults>(
    `protocol-defaults-${name}`,
    () => protocolApi.getDefaults(name).then((res) => res.data),
    { enabled: !!name }
  )
}
