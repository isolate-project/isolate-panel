import { useQuery } from './useQuery'
import { systemApi } from '../api/endpoints'

// Get system resources (RAM/CPU)
export function useSystemResources() {
  return useQuery(
    'system-resources',
    () => systemApi.resources().then((res) => res.data),
    {
      refetchInterval: 5000, // Poll every 5 seconds
    }
  )
}

// Get system health
export function useSystemHealth() {
  return useQuery(
    'system-health',
    () => systemApi.health().then((res) => res.data),
    {
      refetchInterval: 30000, // Poll every 30 seconds
    }
  )
}
