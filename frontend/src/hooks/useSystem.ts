import { useQuery } from './useQuery'
import { systemApi } from '../api/endpoints'

// Stable fetcher functions (not recreated on each render)
const fetchResources = () => systemApi.resources().then((res) => res.data)
const fetchHealth = () => systemApi.health().then((res) => res.data)

// Get system resources (RAM/CPU) - NO POLLING for now
export function useSystemResources() {
  return useQuery(
    'system-resources',
    fetchResources,
    {
      refetchInterval: undefined, // Disabled polling
    }
  )
}

// Get system health - NO POLLING for now
export function useSystemHealth() {
  return useQuery(
    'system-health',
    fetchHealth,
    {
      refetchInterval: undefined, // Disabled polling
    }
  )
}
