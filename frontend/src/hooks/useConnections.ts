import { useQuery } from './useQuery'
import { systemApi } from '../api/endpoints'

interface ConnectionsData {
  count: number
}

// Stable fetcher function (not recreated on each render)
const fetchConnections = () => systemApi.connections().then((res) => res.data)

// Get active connections count - NO POLLING for now
export function useConnections() {
  const { data, isLoading, error } = useQuery<ConnectionsData>(
    'system-connections',
    fetchConnections,
    {
      refetchInterval: undefined, // Disabled polling
    }
  )

  return {
    count: data?.count ?? 0,
    isLoading,
    error,
  }
}
