import { useQuery } from './useQuery'
import { systemApi } from '../api/endpoints'

interface ConnectionsData {
  count: number
}

// Get active connections count with polling
export function useConnections() {
  const { data, isLoading, error } = useQuery<ConnectionsData>(
    'system-connections',
    () => systemApi.connections().then((res) => res.data),
    {
      refetchInterval: 15000, // Poll every 15 seconds
    }
  )

  return {
    count: data?.count ?? 0,
    isLoading,
    error,
  }
}
